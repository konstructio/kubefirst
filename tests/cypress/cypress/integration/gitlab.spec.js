/// <reference types="cypress" />

context('Window', () => {
  before(() => {
    cy.visit('/users/sign_out')
  })

  it('logs in with root user', () => {
    cy.visit('/')
    cy.get('#user_login')
      .type('root')
    cy.get('#user_password')
      .type(Cypress.env('gitlab_bot_password'))
    cy.get('.gl-button').click()
  })
  
  it('metaphor and gitops repos added to kubefirst gitlab project', () => {
    cy.visit('/kubefirst')
    cy.get('#group-3 > .group-row-contents > .group-text-container > .group-text > .d-flex > [data-testid=group-name]').contains('gitops')
    cy.get('#group-2 > .group-row-contents > .group-text-container > .group-text > .d-flex > [data-testid=group-name]').contains('metaphor')
  })
  
  it('initial commit was successfully pushed to metaphor', () => {
    cy.visit('/kubefirst/metaphor')
    cy.get(':nth-child(1) > .d-none > .gl-link').contains('initial kubefirst commit')
  })

  it('initial commit was successfully pushed to gitops', () => {
    cy.visit('/kubefirst/gitops')
    cy.get(':nth-child(2) > .d-none > .gl-link').contains('initial kubefirst commit')
  })

  it('initial commit was successfully pushed to metaphor', () => {
    cy.visit('/kubefirst/metaphor')
    cy.get(':nth-child(1) > .d-none > .gl-link').contains('initial kubefirst commit')
  })

  it('initial merge request was opened against gitops repo with 5 files changed', () => {
    cy.visit('/kubefirst/gitops/-/merge_requests/1')
    cy.get('#diffs-tab > a > .badge').contains('5')
  })

})
