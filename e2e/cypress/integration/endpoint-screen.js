const TEST_ID = Math.random().toString(36).slice(2, 7);
const TEST_ENDPOINT_URL = `${location.protocol}//${location.host}/to/${TEST_ID}`;

describe("Endpoint screen", () => {
  beforeEach(() => {
    cy.visit(`/${TEST_ID}`);
  });

  describe("Unique URL", () => {
    it("should be visible", () => {
      cy.location().then((location) => {
        cy.get('[data-test="unique-endpoint-url').should(
          "contain.text",
          TEST_ENDPOINT_URL
        );
      });
    });
  });

  describe("Requests", () => {
    it("should show a text message if no requests are found", () => {
      cy.get('[data-test="requests').should(
        "contain.text",
        "Waiting for requests..."
      );
    });

    it("should display new requests in real-time", () => {
      const requestBody = "Hello World!";
      cy.exec(`curl -X POST -d '${requestBody}' ${TEST_ENDPOINT_URL}`).then(
        () => {
          cy.get('[data-test="requests').should("contain.text", requestBody);
        }
      );
    });
  });
});
