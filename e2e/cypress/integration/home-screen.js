describe("Home screen", () => {
  beforeEach(() => {
    cy.visit("/");
  });

  describe("Create endpoint button", () => {
    it("should be visible", () => {
      cy.get('button[data-test="create-endpoint-button').should("be.visible");
    });

    it("should redirect to the endpoint screen", () => {
      cy.get('button[data-test="create-endpoint-button').click();
      cy.get('[data-test="unique-endpoint-url').should("be.visible");
    });
  });
});
