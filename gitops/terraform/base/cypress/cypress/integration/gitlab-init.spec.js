/// <reference types="cypress" />

context('Window', () => {
  before(() => {
    cy.visit('/')
  })

  it('logs in with root user', () => {
    cy.get('#user_login')
      .type(Cypress.env('gitlab_bot_username_before'))
    cy.get('#user_password')
      .type(Cypress.env('gitlab_bot_password'))
    cy.get('.gl-button').click()
  })
  
  it('sets up a personal access token', () => {
    cy.visit('/-/profile/personal_access_tokens')
    cy.get('#personal_access_token_name').type('kubefirst')
    cy.get('#personal_access_token_scopes_api').check() 
    cy.get('#personal_access_token_scopes_write_repository').check()
    cy.get('#personal_access_token_scopes_write_registry').check()
    cy.get('#new_personal_access_token > .gl-mt-3 > .gl-button').click()
    cy.get('#created-personal-access-token').then(elem => {
      // elem is the underlying Javascript object targeted by the .get() command.
      const token = Cypress.$(elem).val();
      cy.writeFile('../.gitlab-bot-access-token', token)
    })
  })

  it('gets the runner registration token', () => {
    cy.visit('/admin/runners')
    cy.get('[data-testid=eye-icon] > use').click()
    cy.get('[data-testid=registration-token] > span').then(elem => {
      // elem is the underlying Javascript object targeted by the .get() command.
      const token = Cypress.$(elem).text();
      cy.writeFile('../.gitlab-runner-registration-token', token.trim())
    })
  })

})
