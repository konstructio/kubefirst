let argocdPassword = "" //TODO: wire in cli

describe('argocd', () => {
  it('has 32 synced applications', () => {
    cy.visit('https://argocd.kubefirst.dev')
    cy.contains('Username').click().type('admin')
    cy.contains('Password').click().type(argocdPassword)
    cy.get('.login__form-row > .argo-button').click()
    cy.contains('32')
    cy.contains('Items per page').click()
    cy.get('.opened > ul > [qe-id="undefined-all"]').click()
    cy.screenshot()

  })
})