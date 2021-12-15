"use strict";
test("query params", () => {
  const url = new URL(
    "https://feeder-sync.nononsenseapps.workers.dev/api/readmark?since=1639493118051"
  );
  const since = url.searchParams.get("since");
  const sinceInt = parseInt(since);
  expect(sinceInt).toBe(1639493118051);
});
