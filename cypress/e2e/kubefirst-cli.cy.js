describe('template spec', () => {
  it('passes', () => {
    cy.exec('kubefirst k3d root-credentials').then((result) => {
      cy.log(result.stdout);
    })
  })
})