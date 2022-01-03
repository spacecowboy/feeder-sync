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
          // TODO version api
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
      const chainId = env.chains.newUniqueId();
      const syncChain = env.chains.get(chainId);

      // Forward to the Durable Object
      const newUrl = new URL(request.url);
      newUrl.pathname = "/join";

      return await syncChain.fetch(newUrl, request);
    }
    case "join":
    case "devices":
    case "readmark":
    case "ereadmark":
    case "feeds": {
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
  deviceList: Map<number, string> = new Map();
  syncCode: string | DurableObjectId;
  currentFeedsETag: string;

  constructor(state: DurableObjectState, env: unknown) {
    this.storage = state.storage;
    this.env = env;
    this.sessions = [];
    this.syncCode = state.id;
    this.currentFeedsETag = emptyETag;

    // We keep track of the last-seen message's timestamp just so that we can assign monotonically
    // increasing timestamps even if multiple messages arrive simultaneously (see below). There's
    // no need to store this to disk since we assume if the object is destroyed and recreated, much
    // more than a millisecond will have gone by.
    this.lastTimestamp = 0;

    // Keep all read marks in memory. This way it is easy to implement FIFO
    this.initialization = this._initialize();
  }

  async _initialize(): Promise<void> {
    const maybeFeeds = await this.storage.get("feeds");
    if (maybeFeeds) {
      this.currentFeedsETag = etagValue((maybeFeeds as GetFeedsResponse).hash);
    }

    const maybeDevices = await this.storage.get("deviceList");
    if (maybeDevices) {
      this.deviceList = maybeDevices as Map<number, string>;
    }

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
    const url = new URL(request.url);
    const path = url.pathname.slice(1).split("/");

    // This only waits first time and all flows enter through here
    await this.initialization;

    const deviceIdString = request.headers.get("X-FEEDER-DEVICE-ID");
    if (deviceIdString) {
      const deviceId = parseInt(deviceIdString);
      const deviceRegistered = this.deviceList.has(deviceId);
      if (!deviceRegistered && url.pathname !== "/join") {
        return new Response("Device not registered", { status: 400 });
      }
    } else if (url.pathname !== "/join") {
      return new Response("Missing Device ID", { status: 400 });
    }

    return await handleErrors(request, async () => {
      switch (path[0]) {
        case "readmark": {
          if (path[1]) {
            return new Response(null, { status: 404 });
          }
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
        case "ereadmark": {
          if (path[1]) {
            return new Response(null, { status: 404 });
          }
          switch (request.method) {
            case "GET": {
              // since query param
              const since = url.searchParams.get("since");
              if (typeof since === "string") {
                return await this.getEncryptedReadRest(parseInt(since));
              }
              return new Response(null, { status: 400 });
            }
            case "POST": {
              const data = JSON.parse(await request.text());
              return await this.markAsEncryptedReadRest(data);
            }
            default:
              return new Response(null, { status: 405 });
          }
        }
        case "join": {
          // Create also ends up here
          if (path[1]) {
            return new Response(`Not found: ${url.pathname}`, { status: 404 });
          }
          if (request.method != "POST") {
            return new Response(null, { status: 405 });
          }
          const data = JSON.parse(await request.text()) as
            | CreateRequest
            | JoinRequest;
          return await this.joinRest(data.deviceName);
        }
        case "devices": {
          if (!path[1]) {
            if (request.method != "GET") {
              return new Response(null, { status: 405 });
            }
            return await this.getDevicesRest(true);
          }
          if (path[2]) {
            return new Response(`Not found: ${url.pathname}`, { status: 404 });
          }
          if (request.method != "DELETE") {
            return new Response(null, { status: 405 });
          }
          const deviceId = parseInt(path[1]);
          const result = await this.removeDevice(deviceId);
          if (result) {
            return await this.getDevicesRest(false);
          } else {
            new Response(`No such device registered: ${deviceId}`, {
              status: 404,
            });
          }
        }
        case "feeds": {
          if (path[1]) {
            return new Response(null, { status: 404 });
          }
          switch (request.method) {
            case "GET": {
              const etag = request.headers.get("If-None-Match");
              return await this.getFeeds(etag);
            }
            case "POST": {
              const ifMatchHash = request.headers.get("If-Match");
              const data = JSON.parse(await request.text());
              return await this.updateFeeds(ifMatchHash, data);
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

  async markAsEncryptedReadRest(
    data: SendEncryptedReadMarkBulkRequest
  ): Promise<Response> {
    if (data.items.length > this.maxReadMarks) {
      return new Response("U 2 phat", { status: 400 });
    }

    for (const mark of data.items) {
      this.lastTimestamp = Math.max(Date.now(), this.lastTimestamp + 1);
      const readMark: EncryptedReadMarkMessage = {
        timestamp: this.lastTimestamp,
        encrypted: mark.encrypted,
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

    const response: GetReadResponse = {
      readMarks: marks,
    };

    const headers = new Headers();
    // Clients should cache empty responses - but not with content
    if (marks.length == 0) {
      headers.set("Cache-Control", "private, max-age=10");
    } else {
      headers.set("Cache-Control", "private, max-age=0");
    }
    return new Response(JSON.stringify(response), {
      status: 200,
      headers: headers,
    });
  }

  async getEncryptedReadRest(since: number): Promise<Response> {
    // No need for pagination - it's sorta built in to the entire since parameter
    const storage = await this.storage.list({
      // TODO version prefixes
      // TODO migrate to R1_ prefix - delete all R_
      prefix: "R_",
      start: `R_${since}`,
    });

    const marks: EncryptedReadMarkMessage[] = [];
    for (const value of storage.values()) {
      if (typeof value === "string") {
        marks.push(JSON.parse(value));
      }
    }

    const response: GetEncryptedReadResponse = {
      readMarks: marks,
    };

    const headers = new Headers();
    // Clients should cache empty responses - but not with content
    if (marks.length == 0) {
      headers.set("Cache-Control", "private, max-age=10");
    } else {
      headers.set("Cache-Control", "private, max-age=0");
    }
    return new Response(JSON.stringify(response), {
      status: 200,
      headers: headers,
    });
  }

  async joinRest(deviceName: string): Promise<Response> {
    const deviceId = await this.addDevice(deviceName);

    const response: JoinResponse = {
      syncCode: this.syncCode.toString(),
      deviceId: deviceId,
    };

    return new Response(JSON.stringify(response), { status: 200 });
  }

  async addDevice(deviceName: string): Promise<number> {
    // Random 64 bit integer
    const deviceId = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);

    this.deviceList.set(deviceId, deviceName);
    this.storage.put("deviceList", this.deviceList);

    return deviceId;
  }

  async removeDevice(deviceId: number): Promise<boolean> {
    const result = this.deviceList.delete(deviceId);
    this.storage.put("deviceList", this.deviceList);
    return result;
  }

  async getDevicesRest(cacheHeaders: boolean): Promise<Response> {
    const devices: DeviceMessage[] = [];
    this.deviceList.forEach((value, key) => {
      devices.push({
        deviceId: key,
        deviceName: value,
      });
    });

    const response: DeviceListResponse = {
      devices: devices,
    };

    const headers = new Headers();
    if (cacheHeaders) {
      headers.set("Cache-Control", "private, max-age=60");
    }
    return new Response(JSON.stringify(response), {
      status: 200,
      headers: headers,
    });
  }

  /**
   *
   * @param etagValue one of '*', 'W/"hash"', '"hash"'
   */
  matchesCurrentETag(etagValue: string): Boolean {
    if (etagValue === "*") {
      return true;
    }

    if (etagValue === this.currentFeedsETag) {
      return true;
    }

    // No W/ prefix
    return etagValue === this.currentFeedsETag.substring(2);
  }

  async getFeeds(etag: string | null): Promise<Response> {
    const headers = new Headers();
    headers.set("Vary", "X-FEEDER-ID, X-FEEDER-DEVICE-ID");

    if (etag && this.matchesCurrentETag(etag)) {
      // Only Vary headers on 304
      return new Response(null, {
        status: 304,
        headers: headers,
      });
    }

    headers.set("Cache-Control", "private, must-revalidate");
    headers.set("ETag", this.currentFeedsETag);

    if (this.currentFeedsETag === emptyETag) {
      return new Response(null, {
        status: 204,
        headers: headers,
      });
    }

    const feeds = await this.storage.get("feeds");

    if (!feeds) {
      return new Response(null, {
        status: 204,
        headers: headers,
      });
    }

    return new Response(JSON.stringify(feeds), {
      status: 200,
      headers: headers,
    });
  }

  async updateFeeds(
    etag: string | null,
    data: UpdateFeedsRequest
  ): Promise<Response> {
    if (etag) {
      if (!this.matchesCurrentETag(etag)) {
        return new Response("You're out of date", { status: 412 });
      }
    } else {
      if (this.currentFeedsETag !== emptyETag) {
        // Only require if-match if we have some data
        return new Response("Missing If-Match header", { status: 428 });
      }
    }

    const feeds: GetFeedsResponse = {
      hash: data.contentHash,
      encrypted: data.encrypted,
    };

    await this.storage.put("feeds", feeds);
    this.currentFeedsETag = etagValue(feeds.hash);

    const response: UpdateFeedsResponse = {
      hash: feeds.hash,
    };

    return new Response(JSON.stringify(response), {
      status: 200,
    });
  }
}

// https://stackoverflow.com/questions/40031688/javascript-arraybuffer-to-hex
const byteToHex: string[] = [];

for (let n = 0; n <= 0xff; ++n) {
  const hexOctet = n.toString(16).padStart(2, "0");
  byteToHex.push(hexOctet);
}

function hex(arrayBuffer: ArrayBuffer): string {
  const buff = new Uint8Array(arrayBuffer);
  const hexOctets = []; // new Array(buff.length) is even faster (preallocates necessary array size), then use hexOctets[i] instead of .push()

  for (let i = 0; i < buff.length; ++i) {
    hexOctets.push(byteToHex[buff[i]]);
  }

  return hexOctets.join("");
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

type EncryptedReadMarkMessage = {
  timestamp: number;
  encrypted: string;
};

type GetReadResponse = {
  readMarks: ReadMarkMessage[];
};

type GetEncryptedReadResponse = {
  readMarks: EncryptedReadMarkMessage[];
};

type SendReadMarkBulkRequest = {
  items: SendReadMarkRequest[];
};

type SendReadMarkRequest = {
  feedUrl: string;
  articleGuid: string;
};

type SendEncryptedReadMarkBulkRequest = {
  items: SendEncryptedReadMarkRequest[];
};

type SendEncryptedReadMarkRequest = {
  encrypted: string;
};

type CreateRequest = {
  deviceName: string;
};

type JoinRequest = {
  deviceName: string;
};

type JoinResponse = {
  syncCode: string;
  deviceId: number;
};

type DeviceMessage = {
  deviceId: number;
  deviceName: string;
};

type DeviceListResponse = {
  devices: DeviceMessage[];
};

type GetFeedsResponse = {
  hash: number;
  encrypted: string;
};

type UpdateFeedsRequest = {
  contentHash: number;
  encrypted: string;
};

type UpdateFeedsResponse = {
  hash: number;
};

const emptyETag: string = 'W/"0"';

function etagValue(hash: number): string {
  return `W/"${hash}"`;
}
