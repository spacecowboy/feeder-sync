async function handleErrors(
  request: Request,
  func: () => Promise<Response>
): Promise<Response> {
  try {
    return await func();
  } catch (err) {
    if (request.headers.get("Upgrade") == "websocket") {
      const pair = new WebSocketPair();
      pair[1].accept();
      // @ts-ignore
      pair[1].send(JSON.stringify({ type: "ERROR", error: err.stack }));
      pair[1].close(1011, "Uncaught exception during session setup");
      return new Response(null, { status: 101, webSocket: pair[0] });
    } else {
      return new Response(null, { status: 500 });
    }
  }
}

export default {
  async fetch(request: Request, env: EnvBinding): Promise<Response> {
    return await handleErrors(request, async () => {
      // We have received an HTTP request! Parse the URL and route the request.

      // TODO enforce HTTPS

      const url = new URL(request.url);
      const path = url.pathname.slice(1).split("/");

      if (!path[0]) {
        return new Response("You are in the wrong place", {
          headers: { "Content-Type": "text/html;charset=UTF-8" },
        });
      }

      switch (path[0]) {
        case "api":
          // This is a request for `/api/...`, call the API handler.
          return handleApiRequest(path.slice(1), request, env);

        default:
          return new Response(`Not found: ${path[0]}`, { status: 404 });
      }
    });
  },
};

async function handleApiRequest(
  path: string[],
  request: Request,
  env: EnvBinding
) {
  if (!path[0]) {
    return new Response("Missing path", { status: 404 });
  }

  switch (path[0]) {
    case "create": {
      if (request.method != "POST") {
        return new Response("Method not allowed", { status: 405 });
      }
      const id = env.chains.newUniqueId();
      return new Response(id.toString(), {
        headers: { "Access-Control-Allow-Origin": "*" },
      });
    }
    case "connect": {
      if (request.method != "GET") {
        return new Response("Method not allowed", { status: 405 });
      }
      if (!path[1]) {
        return new Response("Missing id in path", { status: 400 });
      }

      // TODO
      // request.headers.get('Authorization');
      const name = path[1];

      let id;
      if (name.match(/^[0-9a-f]{64}$/)) {
        id = env.chains.idFromString(name);
      } else {
        return new Response("Invalid ID", { status: 404 });
      }

      const syncChain = env.chains.get(id);

      // Forward rest of chain to the Durable Object
      const newUrl = new URL(request.url);
      newUrl.pathname = "/" + path.slice(2).join("/");

      return syncChain.fetch(newUrl, request);
    }
    default:
      return new Response(`Not found: ${path[0]}`, { status: 404 });
  }
}

export class SyncChain {
  lastTimestamp: number;
  sessions: Session[];
  env: unknown;
  storage: DurableObjectStorage;

  constructor(state: DurableObjectState, env: unknown) {
    this.storage = state.storage;
    this.env = env;
    this.sessions = [];

    // We keep track of the last-seen message's timestamp just so that we can assign monotonically
    // increasing timestamps even if multiple messages arrive simultaneously (see below). There's
    // no need to store this to disk since we assume if the object is destroyed and recreated, much
    // more than a millisecond will have gone by.
    this.lastTimestamp = 0;
  }

  async fetch(request: Request): Promise<Response> {
    return await handleErrors(request, async () => {
      const url = new URL(request.url);

      switch (url.pathname) {
        case "/websocket": {
          // A client is trying to establish a new WebSocket session.
          if (request.headers.get("Upgrade") != "websocket") {
            return new Response("expected websocket upgrade", { status: 400 });
          }

          // Get the client's IP address for use with the rate limiter.
          const ip = request.headers.get("CF-Connecting-IP");

          // To accept the WebSocket request, we create a WebSocketPair (which is like a socketpair,
          // i.e. two WebSockets that talk to each other), we return one end of the pair in the
          // response, and we operate on the other end. Note that this API is not part of the
          // Fetch API standard; unfortunately, the Fetch API / Service Workers specs do not define
          // any way to act as a WebSocket server today.
          const pair = new WebSocketPair();

          // We're going to take pair[1] as our end, and return pair[0] to the client.
          await this.handleSession(pair[1], ip);

          // Now we return the other end of the pair to the client.
          return new Response(null, { status: 101, webSocket: pair[0] });
        }
        default:
          return new Response(`Not found: ${url.pathname}`, { status: 404 });
      }
    });
  }

  async handleSession(webSocket: WebSocket, ip: string | null): Promise<void> {
    // Accept our end of the WebSocket. This tells the runtime that we'll be terminating the
    // WebSocket in JavaScript, not sending it elsewhere.
    webSocket.accept();

    // TODO rate limiter

    const session = {
      webSocket,
      dead: false,
    };
    this.sessions.push(session);

    // On "close" and "error" events, remove the WebSocket from the sessions list
    const closeOrErrorHandler = (msg: Event) => {
      session.dead = true;
      this.sessions = this.sessions.filter((member) => member !== session);
    };
    webSocket.addEventListener("close", closeOrErrorHandler);
    webSocket.addEventListener("error", closeOrErrorHandler);
    webSocket.addEventListener("message", async (msg: MessageEvent) => {
      try {
        if (session.dead) {
          // We received a message but marked the session as dead - should never happen but hey
          webSocket.close(1011, "WebSocket broken.");
          return;
        }

        // TODO check rate limit
        let data;

        if (typeof msg.data === "string") {
          data = JSON.parse(msg.data);
        } else {
          webSocket.send(
            JSON.stringify({
              type: "ERROR",
              error: "message data was not string",
            })
          );
          return;
        }

        switch (data.type) {
          case "READ_MARK":
            await this.markAsRead(data, session);
            return;
          case "GET_READ":
            await this.getRead(data, session);
            return;
          default:
            webSocket.send(
              JSON.stringify({
                type: "ERROR",
                error: "Unknown type: " + data.type,
              })
            );
            return;
        }
      } catch (err) {
        // Report any exceptions directly back to the client. As with our handleErrors() this
        // probably isn't what you'd want to do in production, but it's convenient when testing.
        // @ts-ignore
        webSocket.send(JSON.stringify({ type: "ERROR", error: err.stack }));
      }
    });
  }

  async markAsRead(data: ReadMarkMessage, session: Session): Promise<void> {
    // Add timestamp. Here's where this.lastTimestamp comes in -- if we receive a bunch of
    // messages at the same time (or if the clock somehow goes backwards????), we'll assign
    // them sequential timestamps, so at least the ordering is maintained.
    this.lastTimestamp = Math.max(Date.now(), this.lastTimestamp + 1);
    data.timestamp = this.lastTimestamp;
    const dataStr = JSON.stringify(data);

    // Save message.
    // TODO TTL metadata
    // TODO TTL different in prod vs dev
    // const suffix = new Date(data.timestamp).toISOString();
    const suffix = data.timestamp.toString();
    const key = `READMARK_${suffix}`;
    await this.storage.put(key, dataStr);

    // TODO send latest timestamp to self

    // Broadcast the message to all other WebSockets.
    this.broadcast(dataStr, session);
  }

  broadcast(data: string, senderSession: Session): void {
    // Update sessions list in case any of the sessions are dead
    this.sessions = this.sessions.filter((session) => {
      // Don't send to yourself
      if (session === senderSession) {
        return true;
      }
      try {
        session.webSocket.send(data);
        return true;
      } catch (err) {
        // Whoops, this connection is dead. Mark it as such and remove it
        session.dead = true;
        return false;
      }
    });
  }

  async getRead(data: GetReadMessage, session: Session): Promise<void> {
    const since = data.since;
    // TODO pagination
    /*

interface DurableObjectListOptions {
  start?: string;
  end?: string;
  prefix?: string;
  reverse?: boolean;
  limit?: number;
  allowConcurrency?: boolean;
  noCache?: boolean;
}
    */
    const storage = await this.storage.list({
      prefix: "READMARK_",
      start: `READMARK_${since.toString()}`,
    });
    const values = [...storage.values()];

    values.forEach((value) => {
      if (typeof value === "string") {
        session.webSocket.send(value);
      }
    });
  }
}

type EnvBinding = {
  chains: any;
};

type Session = {
  webSocket: WebSocket;
  dead: boolean;
};

type ReadMarkMessage = {
  timestamp: number;
};

type GetReadMessage = {
  since: number;
};
