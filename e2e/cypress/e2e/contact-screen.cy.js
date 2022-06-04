describe("Contact screen", () => {
  beforeEach(() => {
    cy.visit("/contact");
  });

  describe("Title", () => {
    it("should be correct", () => {
      cy.title().should("eq", `Contact | httphq`);
    });
  });

  describe("Form", () => {
    it("should be functional", () => {
      cy.get('input[name="name"]').type("John Doe");
      cy.get('input[name="email"]').type("john@doe.test");
      cy.get('textarea[name="message"]').type("Hello, World!");
      cy.get('button[data-test="send-form"]').should("be.enabled");
    });
  });
});
