describe('development metaphor', () => {
  it('has all the right content', () => {
    cy.visit('https://metaphor-development.kubefirst.dev')
    cy.contains('0.0.1')
    cy.contains('Running')
    cy.contains('your-first-config')
    cy.contains('your-second-config')
    cy.contains('development secret 1')
    cy.contains('development secret 2')
    cy.screenshot()
  })
})

describe('staging metaphor', () => {
  it('has all the right content', () => {
    cy.visit('https://metaphor-staging.kubefirst.dev')
    cy.contains('0.0.1')
    cy.contains('Running')
    cy.contains('your-first-config')
    cy.contains('your-second-config')
    cy.contains('staging secret 1')
    cy.contains('staging secret 2')
    cy.screenshot()
  })
})

describe('production metaphor', () => {
  it('has all the right content', () => {
    cy.visit('https://metaphor-production.kubefirst.dev')
    cy.contains('0.0.1')
    cy.contains('Running')
    cy.contains('your-first-config')
    cy.contains('your-second-config')
    cy.contains('production secret 1')
    cy.contains('production secret 2')
    cy.screenshot()
  })
})