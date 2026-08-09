import { test, expect } from "@playwright/test";
import { captureUrl, newEndpointId, requestsUrl } from "./support/harness";

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
    test("a body over the limit is rejected", async ({ request }) => {
      const overOneMebibyte = "a".repeat(1024 * 1024 + 1);

      const response = await request.post(captureUrl(newEndpointId()), {
        data: overOneMebibyte,
        headers: { "Content-Type": "text/plain" },
      });

      expect(response.status()).toBe(413);
    });

    test("a body at the limit is accepted", async ({ request }) => {
      const endpointId = newEndpointId();
      const oneMebibyte = "a".repeat(1024 * 1024);

      const response = await request.post(captureUrl(endpointId), {
        data: oneMebibyte,
        headers: { "Content-Type": "text/plain" },
      });

      expect(response.status()).toBe(200);

      const listing = await request.get(requestsUrl(endpointId));
      const payload = (await listing.json()) as {
        requests: { body: string }[];
      };
      expect(payload.requests[0].body).toHaveLength(oneMebibyte.length);
    });
  });
});
