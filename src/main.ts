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
      pair[1].send(JSON.stringify({ type: "E", error: err.stack }));
      pair[1].close(1011, "Uncaught exception");
      return new Response(null, {
        status: 101,
        webSocket: pair[0],
      });
    } else {
      // @ts-ignore
      return new Response(JSON.stringify({ type: "E", error: err.stack }), {
        status: 500,
      });
    }
  }
}

export default {
  async fetch(request: Request, env: EnvBinding): Promise<Response> {
    return await handleErrors(request, async () => {
      // We have received an HTTP request! Parse the URL and route the request.
      const url = new URL(request.url);

      if (
        "https:" !== url.protocol ||
        "https" !== request.headers.get("x-forwarded-proto")
      ) {
        return new Response("Only https allowed", { status: 400 });
      }

      const path = url.pathname.slice(1).split("/");

      if (!path[0]) {
        return new Response("You are in the wrong place", {
          headers: { "Content-Type": "text/html;charset=UTF-8" },
        });
      }

      switch (path[0]) {
        case "api":
          // This is a request for `/api/...`, call the API handler.
          return await handleApiRequest(path.slice(1), request, env);

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
): Promise<Response> {
  if (!path[0]) {
    return new Response("Missing path", { status: 404 });
  }

  switch (path[0]) {
    case "create": {
      if (request.method != "POST") {
        return new Response("Method not allowed", { status: 405 });
      }
      const result: CreateResponse = {
        id: env.chains.newUniqueId().toString(),
      };
      return new Response(JSON.stringify(result), {
        headers: { "Access-Control-Allow-Origin": "*" },
      });
    }
    case "readmark": {
      const name = request.headers.get("X-FEEDER-ID");
      if (!name) {
        return new Response("Missing ID", { status: 400 });
      }

      let id;
      if (name.match(/^[0-9a-f]{64}$/)) {
        id = env.chains.idFromString(name);
      } else {
        return new Response("Invalid ID", { status: 400 });
      }

      const syncChain = env.chains.get(id);

      // Forward to the Durable Object
      const newUrl = new URL(request.url);
      newUrl.pathname = "/" + path.join("/");

      return await syncChain.fetch(newUrl, request);
    }
    case "connect": {
      if (request.method != "GET") {
        return new Response("Method not allowed", { status: 405 });
      }

      const name = request.headers.get("X-FEEDER-ID");
      if (!name) {
        return new Response("Missing ID", { status: 400 });
      }

      let id;
      if (name.match(/^[0-9a-f]{64}$/)) {
        id = env.chains.idFromString(name);
      } else {
        return new Response("Invalid ID", { status: 400 });
      }

      const syncChain = env.chains.get(id);

      // Forward rest of chain to the Durable Object
      const newUrl = new URL(request.url);
      newUrl.pathname = "/" + path.slice(1).join("/");

      return await syncChain.fetch(newUrl, request);
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
  initialization: Promise<void>;
  readMarkKeys: string[] = [];
  // Should be less than 1000
  maxReadMarks = 900;

  constructor(state: DurableObjectState, env: unknown) {
    this.storage = state.storage;
    this.env = env;
    this.sessions = [];

    // We keep track of the last-seen message's timestamp just so that we can assign monotonically
    // increasing timestamps even if multiple messages arrive simultaneously (see below). There's
    // no need to store this to disk since we assume if the object is destroyed and recreated, much
    // more than a millisecond will have gone by.
    this.lastTimestamp = 0;

    // Keep all read marks in memory. This way it is easy to implement FIFO
    this.initialization = this._initialize();
  }

  async _initialize(): Promise<void> {
    const stuff = await this.storage.list({
      // TODO version prefixes
      // TODO migrate to R1_ prefix - delete all R_
      prefix: "R_",
    });

    this.readMarkKeys = [...stuff.keys()];

    await this.pruneStorage();
  }

  async pruneStorage(): Promise<void> {
    while (this.readMarkKeys.length > this.maxReadMarks) {
      // Delete limits to 128 keys max at a time
      const itemsToDelete = Math.min(
        128,
        this.readMarkKeys.length - this.maxReadMarks
      );
      await this.storage.delete(this.readMarkKeys.slice(0, itemsToDelete));
      this.readMarkKeys = this.readMarkKeys.slice(itemsToDelete);
    }
  }

  async fetch(request: Request): Promise<Response> {
    // This only waits first time and all flows enter through here
    await this.initialization;

    return await handleErrors(request, async () => {
      const url = new URL(request.url);

      switch (url.pathname) {
        case "/readmark": {
          switch (request.method) {
            case "GET": {
              // since query param
              const since = url.searchParams.get("since");
              if (typeof since === "string") {
                return await this.getReadRest(parseInt(since));
              }
              return new Response(null, { status: 400 });
            }
            case "POST": {
              const data = JSON.parse(await request.text());
              return await this.markAsReadRest(data);
            }
            default:
              return new Response(null, { status: 405 });
          }
        }
        default:
          return new Response(`Not found: ${url.pathname}`, { status: 404 });
      }
    });
  }

  async markAsReadRest(data: SendReadMarkBulkRequest): Promise<Response> {
    if (data.items.length > this.maxReadMarks) {
      return new Response("U 2 phat", { status: 400 });
    }

    for (const mark of data.items) {
      this.lastTimestamp = Math.max(Date.now(), this.lastTimestamp + 1);
      const readMark: ReadMarkMessage = {
        timestamp: this.lastTimestamp,
        feedUrl: mark.feedUrl,
        articleGuid: mark.articleGuid,
      };

      // TODO version prefixes
      // TODO migrate to R1_ prefix - delete all R_
      const suffix = readMark.timestamp;
      const key = `R_${suffix}`;
      await this.storage.put(key, JSON.stringify(readMark));
      // And in-memory collection of keys
      this.readMarkKeys.push(key);
    }

    await this.pruneStorage();

    return new Response(JSON.stringify({ timestamp: this.lastTimestamp }), {
      status: 200,
    });
  }

  async getReadRest(since: number): Promise<Response> {
    // No need for pagination - it's sorta built in to the entire since parameter
    const storage = await this.storage.list({
      // TODO version prefixes
      // TODO migrate to R1_ prefix - delete all R_
      prefix: "R_",
      start: `R_${since}`,
    });

    const marks: ReadMarkMessage[] = [];
    for (const value of storage.values()) {
      if (typeof value === "string") {
        marks.push(JSON.parse(value));
      }
    }

    const headers = new Headers();
    // Clients should cache empty responses - but not with content
    if (marks.length == 0) {
      headers.set("Cache-Control", "private, max-age=10");
    } else {
      headers.set("Cache-Control", "private, max-age=0");
    }
    return new Response(JSON.stringify({ readMarks: marks }), {
      status: 200,
      headers: headers,
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
  feedUrl: string;
  articleGuid: string;
};

type SendReadMarkBulkRequest = {
  items: SendReadMarkRequest[];
};

type SendReadMarkRequest = {
  feedUrl: string;
  articleGuid: string;
};

type GetReadMessage = {
  since: number;
};

type CreateResponse = {
  id: string;
};
