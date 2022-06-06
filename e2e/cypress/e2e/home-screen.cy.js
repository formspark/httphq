describe("Home screen", () => {
  beforeEach(() => {
    cy.visit("/");
  });

  describe("Title", () => {
    it("should be correct", () => {
      cy.title().should("eq", `httphq`);
    });
  });

  describe("Create endpoint button", () => {
    it("should be visible", () => {
      cy.get('button[data-test="create-endpoint"]').should("be.visible");
    });

    it("should redirect to the endpoint screen", () => {
      cy.get('button[data-test="create-endpoint"]').click();
      cy.get('[data-test="endpoint-url"]').should("be.visible");
    });
  });
});
