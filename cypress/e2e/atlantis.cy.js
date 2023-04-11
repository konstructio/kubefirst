let username = "kubefirst-demo-bot"
let password = "" //TODO: wire in cli

describe('template spec', () => {
  it('passes', () => {
    cy.visit('https://github.com/' + username)
    cy.get('.Button-label').click()
    cy.contains('Sign in').click()
    cy.get('#login_field').click().type(username)
    cy.get('#password').click().type(password)
    cy.get('.btn').click()
    cy.visit('https://github.com/' + username + '/gitops')
    cy.screenshot()
  })
})