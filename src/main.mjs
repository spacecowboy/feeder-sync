// `handleErrors()` is a little utility function that can wrap an HTTP request handler in a
// try/catch and return errors to the client. You probably wouldn't want to use this in production
// code but it is convenient when debugging and iterating.
async function handleErrors(request, func) {
  try {
    return await func();
  } catch (err) {
    console.log(`error: ${error}, ${error.stack}`);
    if (request.headers.get("Upgrade") == "websocket") {
      // Annoyingly, if we return an HTTP error in response to a WebSocket request, Chrome devtools
      // won't show us the response body! So... let's send a WebSocket response with an error
      // frame instead.
      let pair = new WebSocketPair();
      pair[1].accept();
      pair[1].send(JSON.stringify({ error: err.stack }));
      pair[1].close(1011, "Uncaught exception during session setup");
      return new Response(null, { status: 101, webSocket: pair[0] });
    } else {
      return new Response(err.stack, { status: 500 });
    }
  }
}

// `fetch` isn't the only handler. If your worker runs on a Cron schedule, it will receive calls
// to a handler named `scheduled`, which should be exported here in a similar way.
export default {
  async fetch(request, env) {
    return await handleErrors(request, async () => {
      // We have received an HTTP request! Parse the URL and route the request.

      // TODO enforce HTTPS

      let url = new URL(request.url);
      let path = url.pathname.slice(1).split("/");

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

async function handleApiRequest(path, request, env) {
  if (!path[0]) {
    return new Response("Missing path", { status: 404 });
  }

  // /api/create - returns id of a new sync chain
  // /api/connect/ID/websocket - connects to a specific sync chain (and creates it if missing)
  // /api/markasread - marks an article as read and broadcasts to other units
  // /api/getread - requests a broadcast of items marked as read since <TIMESTAMP>
  switch (path[0]) {
    case "create": {
      if (request.method != "POST") {
        return new Response("Method not allowed", { status: 405 });
      }
      let id = env.chains.newUniqueId();
      return new Response(id.toString(), {
        headers: { "Access-Control-Allow-Origin": "*" },
      });
    }
    case "connect": {
      if (request.method != "GET") {
        return new Response("Method not allowed", { status: 405 });
      }
      if (!path[1]) {
        console.log("Missing id in path");
        return new Response("Missing id in path", { status: 400 });
      }

      // TODO
      // request.headers.get('Authorization');
      let name = path[1];

      let id;
      if (name.match(/^[0-9a-f]{64}$/)) {
        id = env.chains.idFromString(name);
      } else {
        return new Response("Invalid ID", { status: 404 });
      }

      let syncChain = env.chains.get(id);

      // Forward rest of chain to the Durable Object
      let newUrl = new URL(request.url);
      newUrl.pathname = "/" + path.slice(2).join("/");

      return syncChain.fetch(newUrl, request);
    }
    default:
      return new Response(`Not found: ${path[0]}`, { status: 404 });
  }
}

export class SyncChain {
  constructor(state, env) {
    this.storage = state.storage;
    this.env = env;
    this.sessions = [];

    // We keep track of the last-seen message's timestamp just so that we can assign monotonically
    // increasing timestamps even if multiple messages arrive simultaneously (see below). There's
    // no need to store this to disk since we assume if the object is destroyed and recreated, much
    // more than a millisecond will have gone by.
    this.lastTimestamp = 0;
  }

  async fetch(request) {
    return await handleErrors(request, async () => {
      let url = new URL(request.url);

      switch (url.pathname) {
        case "/websocket": {
          // A client is trying to establish a new WebSocket session.
          if (request.headers.get("Upgrade") != "websocket") {
            console.log("Expected websocket upgrade");
            return new Response("expected websocket upgrade", { status: 400 });
          }

          // Get the client's IP address for use with the rate limiter.
          let ip = request.headers.get("CF-Connecting-IP");

          // To accept the WebSocket request, we create a WebSocketPair (which is like a socketpair,
          // i.e. two WebSockets that talk to each other), we return one end of the pair in the
          // response, and we operate on the other end. Note that this API is not part of the
          // Fetch API standard; unfortunately, the Fetch API / Service Workers specs do not define
          // any way to act as a WebSocket server today.
          let pair = new WebSocketPair();

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

  async handleSession(webSocket, ip) {
    // Accept our end of the WebSocket. This tells the runtime that we'll be terminating the
    // WebSocket in JavaScript, not sending it elsewhere.
    webSocket.accept();

    // TODO rate limiter

    let session = {
      webSocket,
      dead: false,
    };
    this.sessions.push(session);

    // On "close" and "error" events, remove the WebSocket from the sessions list
    let closeOrErrorHandler = (evt) => {
      session.dead = true;
      this.sessions = this.sessions.filter((member) => member !== session);
    };
    webSocket.addEventListener("close", closeOrErrorHandler);
    webSocket.addEventListener("error", closeOrErrorHandler);
    webSocket.addEventListener("message", async (msg) => {
      try {
        if (session.dead) {
          // We received a message but marked the session as dead - should never happen but hey
          webSocket.close(1011, "WebSocket broken.");
          return;
        }

        // TODO check rate limit

        /*
        Format of JSON
        
        { type: METHOD, ...}

        where

        { type: markasread, articleId: ARTICLEID }

        { type: getread, since: TIMESTAMP }
        */
        let data = JSON.parse(msg.data);

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
        webSocket.send(JSON.stringify({ type: "ERROR", error: err.stack }));
      }
    });
  }

  async markAsRead(data, session) {
    // Add timestamp. Here's where this.lastTimestamp comes in -- if we receive a bunch of
    // messages at the same time (or if the clock somehow goes backwards????), we'll assign
    // them sequential timestamps, so at least the ordering is maintained.
    this.lastTimestamp = Math.max(Date.now(), this.lastTimestamp + 1);
    data.timestamp = this.lastTimestamp;
    let dataStr = JSON.stringify(data);

    // Save message.
    // TODO TTL metadata
    // TODO TTL different in prod vs dev
    let key = new Date(data.timestamp).toISOString();
    await this.storage.put(key, dataStr, { expirationTtl: 3600 });

    // Broadcast the message to all other WebSockets.
    this.broadcast(dataStr, session);
  }

  broadcast(data, senderSession) {
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

  async getRead(data, session) {
    const since = data.since;

    // TODO binary search
    // TODO prefix them

    /*
    {
  keys: [{ name: "foo", expiration: 1234, metadata: {someMetadataKey: "someMetadataValue"}}],
  list_complete: false,
  cursor: "6Ck1la0VxJ0djhidm1MdX2FyD"
}
*/

    // TODO prefix on read marks
    // TODO what about start option?
    // TODO cursor // list({"cursor": cursor})
    const storage = await this.storage.list({ limit: 100 });
    const values = [...storage.values()];
    session.webSocket.send(
      JSON.stringify({ type: "ERROR", error: JSON.stringify(values) })
    );

    values.forEach((value) => {
      // key is in ISO Date
      // const timestamp = parseInt(value.name)
      // if (timestamp > since) {
      session.webSocket.send(value);
      // }
    });
  }
}
