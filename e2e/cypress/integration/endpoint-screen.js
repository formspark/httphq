const TEST_ID = "test-endpoint";

describe("Endpoint screen", () => {
  beforeEach(() => {
    cy.visit(`/${TEST_ID}`);
  });

  describe("Unique URL", () => {
    it("should be visible", () => {
      cy.location().then((location) => {
        cy.get('[data-test="unique-endpoint-url').should(
          "have.text",
          `${location.protocol}//${location.host}/to/${TEST_ID}`
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
  });
});
