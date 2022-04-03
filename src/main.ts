async function handleErrors(
  request: Request,
  func: () => Promise<Response>
): Promise<Response> {
  try {
    return await func();
  } catch (err) {
    return new Response(JSON.stringify({ type: "E", error: err }), {
      status: 500,
    });
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
          status: 404,
          headers: { "Content-Type": "text/html;charset=UTF-8" },
        });
      }

      switch (path[0]) {
        case "api":
          switch (path[1]) {
            case "v1":
              return await handleApiV1Request(path.slice(2), request, env);
            default:
              return new Response(`Not found: ${path[0]}`, { status: 404 });
          }
        case ".well-known":
          return handleWellKnownRequest(path.slice(1));
        default:
          return new Response(`Not found: ${path[0]}`, { status: 404 });
      }
    });
  },
  async scheduled(
    event: ScheduledEvent,
    env: EnvBinding,
    ctx: ExecutionContext
  ): Promise<void> {
    ctx.waitUntil(handleCronEvent(env));
  },
};

async function handleCronEvent(env: EnvBinding): Promise<void> {
  let cursor = "";

  do {
    const allSyncChains = await getAllSyncChains(env, cursor);

    if (!allSyncChains.success) {
      throw "Failed to get all sync chains";
    }

    cursor = allSyncChains.result_info.cursor;

    for (const meta of allSyncChains.result) {
      const id = env.chains.idFromString(meta.id);
      const syncChain: DurableObjectStub = env.chains.get(id);

      await syncChain.fetch("https://cron/self_destruct_if_old");
    }
  } while (cursor.length > 0);

  return;
}

async function isLatestReadMarkOlderThan90Days(
  readMarkKeys: string[]
): Promise<boolean> {
  // Remove R1_ prefix
  const lastTimestamp = readMarkKeys[readMarkKeys.length - 1].substring(3);
  // Calculate age of that last timestamp until now
  const millisSinceLastUse = Date.now() - parseInt(lastTimestamp);
  const ninetyDays = 1000 * 60 * 60 * 24 * 90;
  return millisSinceLastUse > ninetyDays;
}

async function getAllSyncChains(
  env: EnvBinding,
  cursor: string
): Promise<CloudflareObjectListResponse> {
  const url = `https://api.cloudflare.com/client/v4/accounts/${env.ACCOUNT_ID}/workers/durable_objects/namespaces/${env.NAMESPACE_ID}/objects?cursor=${cursor}`;

  const init = {
    headers: {
      Authorization: `Bearer ${env.TOKEN}`,
      "Content-Type": "application/json",
    },
  };

  const response = await fetch(url, init);
  const jsonResponse: CloudflareObjectListResponse = await response.json();

  return jsonResponse;
}

async function handleWellKnownRequest(path: string[]): Promise<Response> {
  if (path.length == 1 && path[0] === "assetlinks.json") {
    // Copied from file in repo
    const jsonFile = `
    [
      {
        "relation": [
          "delegate_permission/common.handle_all_urls"
        ],
        "target": {
          "namespace": "android_app",
          "package_name": "com.nononsenseapps.feeder",
          "sha256_cert_fingerprints": [
            "1F:36:57:FC:FB:C0:73:DF:5F:EA:C8:65:00:58:0D:17:5A:C4:FD:76:9E:C5:13:23:F8:CC:64:56:AA:CA:F2:BF",
            "C5:EE:FF:22:48:81:35:FF:C2:58:3C:3A:43:B0:53:A1:61:CA:86:98:62:96:1A:B8:53:4F:44:C7:5F:D5:7D:97",
            "43:23:8D:51:2C:1E:5E:B2:D6:56:9F:4A:3A:FB:F5:52:34:18:B8:2E:0A:3E:D1:55:27:70:AB:B9:A9:C9:CC:AB"
          ]
        }
      },
      {
        "relation": [
          "delegate_permission/common.handle_all_urls"
        ],
        "target": {
          "namespace": "android_app",
          "package_name": "com.nononsenseapps.feeder.play",
          "sha256_cert_fingerprints": [
            "1F:36:57:FC:FB:C0:73:DF:5F:EA:C8:65:00:58:0D:17:5A:C4:FD:76:9E:C5:13:23:F8:CC:64:56:AA:CA:F2:BF",
            "C5:EE:FF:22:48:81:35:FF:C2:58:3C:3A:43:B0:53:A1:61:CA:86:98:62:96:1A:B8:53:4F:44:C7:5F:D5:7D:97",
            "AC:75:28:54:1E:6F:FC:7D:AD:2C:C7:AA:52:51:12:31:93:C0:09:2C:5B:52:FC:26:62:9D:0F:73:76:81:9D:58"
          ]
        }
      },
      {
        "relation": [
          "delegate_permission/common.handle_all_urls"
        ],
        "target": {
          "namespace": "android_app",
          "package_name": "com.nononsenseapps.feeder.debug",
          "sha256_cert_fingerprints": [
            "1F:36:57:FC:FB:C0:73:DF:5F:EA:C8:65:00:58:0D:17:5A:C4:FD:76:9E:C5:13:23:F8:CC:64:56:AA:CA:F2:BF",
            "C5:EE:FF:22:48:81:35:FF:C2:58:3C:3A:43:B0:53:A1:61:CA:86:98:62:96:1A:B8:53:4F:44:C7:5F:D5:7D:97"
          ]
        }
      }
    ]    
    `;
    return new Response(jsonFile, {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  }

  return new Response(`Not found: ${path[0]}`, { status: 404 });
}

async function handleApiV1Request(
  path: string[],
  request: Request,
  env: EnvBinding
): Promise<Response> {
  const userAndPass = basicAuthentication(request);
  if (userAndPass == null) {
    return new Response("Missing credentials", { status: 401 });
  }

  if (!verifyCredentials(userAndPass.user, userAndPass.pass)) {
    return new Response("Not authorized", { status: 401 });
  }

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

      return await syncChain.fetch(`${newUrl}`, request);
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

      return await syncChain.fetch(`${newUrl}`, request);
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
  currentDevicesETagNumber: number;
  currentReadETagNumber: number;

  constructor(state: DurableObjectState, env: unknown) {
    this.storage = state.storage;
    this.env = env;
    this.sessions = [];
    this.syncCode = state.id;
    this.currentFeedsETag = emptyETag;
    this.currentDevicesETagNumber = 0;
    this.currentReadETagNumber = 0;

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
      prefix: "R1_",
    });

    this.readMarkKeys = [...stuff.keys()];

    await this.pruneStorage();
  }

  async selfDestructIfOldOrEmpty(): Promise<void> {
    const shouldBeDeleted =
      this.deviceList.size === 0 ||
      this.readMarkKeys.length === 0 ||
      isLatestReadMarkOlderThan90Days(this.readMarkKeys);

    if (shouldBeDeleted) {
      // Once the object shuts down after this it will cease to exist
      await this.storage.deleteAll();
    }
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

    // Temporary migration
    const oldStuff = await this.storage.list({
      prefix: "R_",
    });
    let oldReadMarkKeys = [...oldStuff.keys()];
    while (oldReadMarkKeys.length > 0) {
      // Delete limits to 128 keys max at a time
      const itemsToDelete = Math.min(128, oldReadMarkKeys.length);
      await this.storage.delete(oldReadMarkKeys.slice(0, itemsToDelete));
      oldReadMarkKeys = oldReadMarkKeys.slice(itemsToDelete);
    }
  }

  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);

    // Special case
    if (url.pathname === "cron/self_destruct_if_old") {
      await this.selfDestructIfOldOrEmpty();
      return new Response(null, { status: 200 });
    }

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
              const etag = request.headers.get("If-None-Match");
              if (typeof since === "string") {
                return await this.getReadRest(parseInt(since), etag);
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
              const etag = request.headers.get("If-None-Match");
              if (typeof since === "string") {
                return await this.getEncryptedReadRest(parseInt(since), etag);
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
            const etag = request.headers.get("If-None-Match");
            return await this.getDevicesRest(true, etag);
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
            return await this.getDevicesRest(false, null);
          } else {
            return new Response(`No such device registered: ${deviceId}`, {
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

      const suffix = readMark.timestamp;
      const key = `R1_${suffix}`;
      await this.storage.put(key, JSON.stringify(readMark));
      // And in-memory collection of keys
      this.readMarkKeys.push(key);
    }

    await this.pruneStorage();

    this.currentReadETagNumber = Math.floor(
      Math.random() * Number.MAX_SAFE_INTEGER
    );

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

      const suffix = readMark.timestamp;
      const key = `R1_${suffix}`;
      await this.storage.put(key, JSON.stringify(readMark));
      // And in-memory collection of keys
      this.readMarkKeys.push(key);
    }

    await this.pruneStorage();

    this.currentReadETagNumber = Math.floor(
      Math.random() * Number.MAX_SAFE_INTEGER
    );

    return new Response(JSON.stringify({ timestamp: this.lastTimestamp }), {
      status: 200,
    });
  }

  async getReadRest(since: number, etag: string | null): Promise<Response> {
    if (
      etag &&
      this.matchesCurrentETag(etag, etagValue(this.currentReadETagNumber))
    ) {
      return new Response(null, { status: 304 });
    }
    // No need for pagination - it's sorta built in to the entire since parameter
    const storage = await this.storage.list({
      prefix: "R1_",
      start: `R1_${since}`,
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
    headers.set("Cache-Control", "private, must-revalidate");
    headers.set("ETag", etagValue(this.currentReadETagNumber));

    return new Response(JSON.stringify(response), {
      status: 200,
      headers: headers,
    });
  }

  async getEncryptedReadRest(
    since: number,
    etag: string | null
  ): Promise<Response> {
    if (
      etag &&
      this.matchesCurrentETag(etag, etagValue(this.currentReadETagNumber))
    ) {
      return new Response(null, { status: 304 });
    }
    // No need for pagination - it's sorta built in to the entire since parameter
    const storage = await this.storage.list({
      prefix: "R1_",
      start: `R1_${since}`,
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
    headers.set("Cache-Control", "private, must-revalidate");
    headers.set("ETag", etagValue(this.currentReadETagNumber));

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
    this.currentDevicesETagNumber = Math.floor(
      Math.random() * Number.MAX_SAFE_INTEGER
    );

    return deviceId;
  }

  async removeDevice(deviceId: number): Promise<boolean> {
    const result = this.deviceList.delete(deviceId);
    this.storage.put("deviceList", this.deviceList);
    this.currentDevicesETagNumber = Math.floor(
      Math.random() * Number.MAX_SAFE_INTEGER
    );
    return result;
  }

  async getDevicesRest(
    cacheHeaders: boolean,
    etag: string | null
  ): Promise<Response> {
    if (
      etag &&
      this.matchesCurrentETag(etag, etagValue(this.currentDevicesETagNumber))
    ) {
      return new Response(null, { status: 304 });
    }

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
      headers.set("Cache-Control", "private, must-revalidate");
      headers.set("ETag", etagValue(this.currentDevicesETagNumber));
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
  matchesCurrentETag(etagValue: string, currentEtag: string): boolean {
    if (etagValue === "*") {
      return true;
    }

    if (etagValue === currentEtag) {
      return true;
    }

    // No W/ prefix
    return etagValue === currentEtag.substring(2);
  }

  async getFeeds(etag: string | null): Promise<Response> {
    if (etag && this.matchesCurrentETag(etag, this.currentFeedsETag)) {
      return new Response(null, {
        status: 304,
      });
    }

    const headers = new Headers();
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
      if (!this.matchesCurrentETag(etag, this.currentFeedsETag)) {
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

type EnvBinding = {
  chains: DurableObjectNamespace;
  ACCOUNT_ID: string;
  TOKEN: string;
  NAMESPACE_ID: string;
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

const emptyETag = 'W/"0"';

function etagValue(hash: number): string {
  return `W/"${hash}"`;
}

// Only used to prevent random bypassers from scanning the API surface
const HARDCODED_USER = "feeder_user";
const HARDCODED_PASSWORD = "feeder_secret_1234";

function verifyCredentials(user: string, pass: string): boolean {
  if (HARDCODED_USER !== user) {
    return false;
  }

  if (HARDCODED_PASSWORD !== pass) {
    return false;
  }

  return true;
}

type UserAndPassword = {
  user: string;
  pass: string;
};

/**
 * Parse HTTP Basic Authorization value.
 */
function basicAuthentication(request: Request): UserAndPassword | null {
  const authorization = request.headers.get("Authorization");

  if (authorization == null) {
    return null;
  }

  const [scheme, encoded] = authorization.split(" ");

  // The Authorization header must start with Basic, followed by a space.
  if (!encoded || scheme !== "Basic") {
    return null;
  }

  // Decodes the base64 value and performs unicode normalization.
  // @see https://datatracker.ietf.org/doc/html/rfc7613#section-3.3.2 (and #section-4.2.2)
  // @see https://dev.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/String/normalize
  const buffer = Uint8Array.from(atob(encoded), (character) =>
    character.charCodeAt(0)
  );
  const decoded = new TextDecoder().decode(buffer).normalize();

  // The username & password are split by the first colon.
  //=> example: "username:password"
  const index = decoded.indexOf(":");

  // The user & password are split by the first colon and MUST NOT contain control characters.
  // @see https://tools.ietf.org/html/rfc5234#appendix-B.1 (=> "CTL = %x00-1F / %x7F")
  // eslint-disable-next-line no-control-regex
  if (index === -1 || /[\0-\x1F\x7F]/.test(decoded)) {
    return null;
  }

  return {
    user: decoded.substring(0, index),
    pass: decoded.substring(index + 1),
  };
}

type CloudflareObjectListResponseResult = {
  hasStoredData: boolean;
  id: string;
};

type CloudflareObjectListResponse = {
  result: CloudflareObjectListResponseResult[];
  success: boolean;
  //errors: any[];
  //messages: any[];
  result_info: {
    count: number;
    cursor: string;
  };
};
