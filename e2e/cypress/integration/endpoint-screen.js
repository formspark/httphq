describe("Endpoint screen", () => {
  let testId = "";
  let testEndpointPath = "";
  let testEndpointUrl = "";

  beforeEach(() => {
    testId = Math.random().toString(36).slice(2, 7);
    testEndpointPath = `/to/${testId}`;
    testEndpointUrl = `${location.protocol}//${location.host}${testEndpointPath}`;

    cy.visit(`/${testId}`);
  });

  describe("Title", () => {
    it("should be correct", () => {
      cy.title().should("eq", `${testId} | httphq`);
    });
  });

  describe("Endpoint URL", () => {
    it("should be visible", () => {
      cy.get('[data-test="endpoint-url"]').should(
        "contain.text",
        testEndpointUrl
      );
    });
  });

  describe("cURL example", () => {
    it("should be visible", () => {
      cy.get('[data-test="curl-example"]').should(
        "contain.text",
        `curl -X POST -d 'Hello, World!' ${testEndpointUrl}`
      );
    });

    it("should be possible to send a request", () => {
      cy.get('[data-test="curl-send"]').click();
      cy.get('[data-test="requests"]').should(
        "not.contain.text",
        "Waiting for requests"
      );
    });
  });

  describe("Requests", () => {
    it("should show a waiting message if no requests are found", () => {
      cy.get('[data-test="requests"]').should(
        "contain.text",
        "Waiting for requests"
      );
    });

    it("should not show a waiting message if some requests are found", () => {
      cy.exec(`curl -X POST -d 'Hello, World!' ${testEndpointUrl}`).then(() => {
        cy.get('[data-test="requests"]').should(
          "not.contain.text",
          "Waiting for requests"
        );
      });
    });

    it("should display new requests in real-time", () => {
      const requestBody = "Hello, World!";
      cy.exec(`curl -X POST -d '${requestBody}' ${testEndpointUrl}`).then(
        () => {
          cy.get('[data-test="requests"]').should("contain.text", requestBody);
        }
      );
    });

    it("should display the request details", () => {
      cy.exec(`curl -X POST -d 'Hello, World!' ${testEndpointUrl}`).then(() => {
        cy.get('[data-test="request-details"]').contains(/now|seconds? ago/);
        cy.get('[data-test="request-details"]').contains("127.0.0.1");
        cy.get('[data-test="request-details"]').contains("POST");
        cy.get('[data-test="request-details"]').contains(testEndpointPath);
      });
    });

    it("should display the request headers", () => {
      cy.exec(`curl -X POST -d 'Hello, World!' ${testEndpointUrl}`).then(() => {
        cy.get('[data-test="request-headers"]').contains("Content-Type");
      });
    });

    it("should display the request body", () => {
      const requestBody = "Hello, World!";
      cy.exec(`curl -X POST -d '${requestBody}' ${testEndpointUrl}`).then(
        () => {
          cy.get('[data-test="request-body"]').contains(requestBody);
        }
      );
    });

    it("should be possible to filter based on the request body", () => {
      const requestBody = "Hello, World!";
      cy.exec(`curl -X POST -d '${requestBody}' ${testEndpointUrl}`).then(
        () => {
          cy.get('[data-test="requests"]').should("contain.text", requestBody);

          cy.get('[data-test="search-input"]').clear().type("Hello");
          cy.get('[data-test="search-results"]').should(
            "contain.text",
            "1 result"
          );
          cy.get('[data-test="requests"]').should("contain.text", requestBody);

          cy.get('[data-test="search-input"]').clear().type("Test");
          cy.get('[data-test="search-results"]').should(
            "contain.text",
            "0 results"
          );
          cy.get('[data-test="requests"]').should(
            "not.contain.text",
            requestBody
          );
        }
      );
    });

    it("should be possible to filter based on the request headers", () => {
      const requestHeaderKey = "A-Test";
      const requestHeaderValue = "Hello, World!";

      cy.exec(
        `curl -X POST -H '${requestHeaderKey}: ${requestHeaderValue}' ${testEndpointUrl}`
      ).then(() => {
        cy.get('[data-test="requests"]').should(
          "contain.text",
          requestHeaderKey
        );
        cy.get('[data-test="requests"]').should(
          "contain.text",
          requestHeaderValue
        );

        // Positive key search
        cy.get('[data-test="search-input"]').clear().type("A-");
        cy.get('[data-test="search-results"]').should(
          "contain.text",
          "1 result"
        );
        cy.get('[data-test="requests"]').should(
          "contain.text",
          requestHeaderKey
        );
        cy.get('[data-test="requests"]').should(
          "contain.text",
          requestHeaderValue
        );

        // Negative key search
        cy.get('[data-test="search-input"]').clear().type("B-");
        cy.get('[data-test="search-results"]').should(
          "contain.text",
          "0 results"
        );
        cy.get('[data-test="requests"]').should(
          "not.contain.text",
          requestHeaderKey
        );
        cy.get('[data-test="requests"]').should(
          "not.contain.text",
          requestHeaderValue
        );

        // Positive value search
        cy.get('[data-test="search-input"]').clear().type("Hello");
        cy.get('[data-test="search-results"]').should(
          "contain.text",
          "1 result"
        );
        cy.get('[data-test="requests"]').should(
          "contain.text",
          requestHeaderKey
        );
        cy.get('[data-test="requests"]').should(
          "contain.text",
          requestHeaderValue
        );

        // Negative value search
        cy.get('[data-test="search-input"]').clear().type("Not");
        cy.get('[data-test="search-results"]').should(
          "contain.text",
          "0 results"
        );
        cy.get('[data-test="requests"]').should(
          "not.contain.text",
          requestHeaderKey
        );
        cy.get('[data-test="requests"]').should(
          "not.contain.text",
          requestHeaderValue
        );
      });
    });

    it("should show a deletion disclaimer", () => {
      cy.contains("⚠️️️ Requests are deleted after 4 hours");
    });

    it("should be possible to delete all requests", () => {
      cy.exec(`curl -X POST -d 'Hello, World!' ${testEndpointUrl}`).then(() => {
        cy.exec(`curl -X POST -d 'Hello, World!' ${testEndpointUrl}`).then(
          () => {
            cy.get('[data-test="delete-requests"]').click();
            cy.get('[data-test="requests"]').should(
              "contain.text",
              "Waiting for requests"
            );
          }
        );
      });
    });

    it("should be possible to delete a specific request", () => {
      cy.exec(`curl -X POST -d 'Hello, World!' ${testEndpointUrl}`).then(() => {
        cy.request("POST", testEndpointUrl, { message: "Hello, World!" }).then(
          (response) => {
            const uuid = response.headers["httphq-request-uuid"];
            cy.get(`#request-${uuid}`).should("exist");
            cy.get(`#request-${uuid}`).within(() => {
              cy.get('[data-test="delete-request"]').click({ force: true });
            });
            cy.get(`#request-${uuid}`).should("not.exist");
          }
        );
      });
    });
  });
});
