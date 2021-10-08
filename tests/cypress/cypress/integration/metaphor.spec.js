/// <reference types="cypress" />

context('Window', () => {
  it('metaphor development', () => {
    cy.visit("https://metaphor-development." + (Cypress.env('AWS_HOSTED_ZONE_NAME')))
    cy.get('body > :nth-child(2) > :nth-child(2)').contains('app_name: metaphor')
    cy.get(':nth-child(3) > :nth-child(2)').contains('SECRET_ONE: development secret')
    cy.get(':nth-child(4) > :nth-child(2)').contains('CONFIG_ONE: your-first-config')
  })
  it('metaphor staging', () => {
    cy.visit("https://metaphor-staging." + (Cypress.env('AWS_HOSTED_ZONE_NAME')))
    cy.get('body > :nth-child(2) > :nth-child(2)').contains('app_name: metaphor')
    cy.get(':nth-child(3) > :nth-child(2)').contains('SECRET_ONE: staging secret')
    cy.get(':nth-child(4) > :nth-child(2)').contains('CONFIG_ONE: your-first-config')
  })
  it('metaphor production', () => {
    cy.visit("https://metaphor-production." + (Cypress.env('AWS_HOSTED_ZONE_NAME')))
    cy.get('body > :nth-child(2) > :nth-child(2)').contains('app_name: metaphor')
    cy.get(':nth-child(3) > :nth-child(2)').contains('SECRET_ONE: production secret')
    cy.get(':nth-child(4) > :nth-child(2)').contains('CONFIG_ONE: your-first-config')
  })
})
