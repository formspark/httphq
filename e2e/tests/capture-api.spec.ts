import { test, expect } from "@playwright/test";
import { captureUrl, listRequests, newEndpointId } from "./support/harness";

/**
 * What only a real socket can show. The Go suite drives the same routes through
 * the router in memory, which is enough for everything a handler decides; the
 * limits enforced by the server around the handler need a client on the wire to
 * observe.
 */
test.describe("Capture API", () => {
  test.describe("Body limit", () => {
    // A shared instance writes to one disk, so an unbounded body is one
    // caller's ability to fill it for everyone.
    //
    // The server answers 413 and closes without draining the rest of the body,
    // so a client still writing sees the connection go instead of the reply.
    // Both are the same refusal and which one lands depends on how far the
    // write had got, so the assertion that holds either way is that nothing
    // was stored.
    test("a body over the limit is rejected", async ({ request }) => {
      const endpointId = newEndpointId();
      const overOneMebibyte = "a".repeat(1024 * 1024 + 1);

      const outcome = await request
        .post(captureUrl(endpointId), {
          data: overOneMebibyte,
          headers: { "Content-Type": "text/plain" },
        })
        .then((response): number | string => response.status())
        .catch(() => "connection closed");

      expect([413, "connection closed"]).toContain(outcome);

      expect((await listRequests(request, endpointId)).requests).toHaveLength(
        0,
      );
    });

    test("a body at the limit is accepted", async ({ request }) => {
      const endpointId = newEndpointId();
      const oneMebibyte = "a".repeat(1024 * 1024);

      const response = await request.post(captureUrl(endpointId), {
        data: oneMebibyte,
        headers: { "Content-Type": "text/plain" },
      });

      expect(response.status()).toBe(200);

      const listing = await listRequests(request, endpointId);
      expect(listing.requests[0].body).toHaveLength(oneMebibyte.length);
    });
  });
});
