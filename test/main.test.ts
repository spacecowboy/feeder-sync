test('query params', () => {
    const url = new URL("https://feeder-sync.nononsenseapps.workers.dev/api/readmark?since=1639493118051");

    const since = url.searchParams.get("since");
    const sinceInt = parseInt(since)

    expect(sinceInt).toBe(1639493118051);
  });

test('interpolation', () => {
    const lastTimestamp = Date.now();

    const readMark = {
      timestamp: lastTimestamp,
      feedUrl: "foo",
      articleGuid: "bar",
    };

    const suffix = readMark.timestamp;
    const key = `R_${suffix}`;

    expect(key).toBe("R_123");
});