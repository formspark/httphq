const TEST_ID = Math.random().toString(36).slice(2, 7);
const TEST_ENDPOINT_PATH = `/to/${TEST_ID}`;
const TEST_ENDPOINT_URL = `${location.protocol}//${location.host}${TEST_ENDPOINT_PATH}`;

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
    it("should show a waiting message if no requests are found", () => {
      cy.get('[data-test="requests').should(
        "contain.text",
        "Waiting for requests..."
      );
    });

    it("should not show a waiting message if some requests are found", () => {
      cy.exec(`curl -X POST -d 'Hello World!' ${TEST_ENDPOINT_URL}`).then(
        () => {
          cy.get('[data-test="requests').should(
            "not.contain.text",
            "Waiting for requests..."
          );
        }
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

    it("should display the request details", () => {
      cy.exec(`curl -X POST -d 'Hello World!' ${TEST_ENDPOINT_URL}`).then(
        () => {
          cy.get('[data-test="request-details').contains(/now|seconds? ago/);
          cy.get('[data-test="request-details').contains("127.0.0.1");
          cy.get('[data-test="request-details').contains("POST");
          cy.get('[data-test="request-details').contains(TEST_ENDPOINT_PATH);
        }
      );
    });

    it("should display the request headers", () => {
      cy.exec(`curl -X POST -d 'Hello World!' ${TEST_ENDPOINT_URL}`).then(
        () => {
          cy.get('[data-test="request-headers').contains("Content-Type");
        }
      );
    });

    it("should display the request body", () => {
      const requestBody = "Hello World!";
      cy.exec(`curl -X POST -d '${requestBody}' ${TEST_ENDPOINT_URL}`).then(
        () => {
          cy.get('[data-test="request-body').contains(requestBody);
        }
      );
    });
  });
});
