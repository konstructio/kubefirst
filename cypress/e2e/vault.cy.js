var vaultRootToken = "" //TODO: wire in cli

describe('vault', () => {
  it('lets you log in with root user to get kbot password', () => {
      cy.visit('/')
      cy.get('#ember50').type(vaultRootToken)
      cy.get('#auth-submit').click()
      cy.wait(1000)
      cy.screenshot()
      //cy.visit('/ui/vault/secrets/users/show/kbot').wait(3000)
      cy.contains('users').click()
      cy.contains('kbot').click()
      cy.get('#icon-ember201').click()
      cy.wait(500)
      cy.get('.masked-value').then(elem => {
        // elem is the underlying Javascript object targeted by the .get() command.
        cy.log(elem)
        const token = Cypress.$(elem).val();
        cy.log(token)
        console.log(token)
        cy.writeFile('./vault-root-token', token)
      })
      
      // // cy.wait(1000)
      // // cy.screenshot()
  })
})