import { describe, test, expect } from 'vitest'
import { wordsToAgeRecipient, createIdentityFromWords } from './backupKey'

describe('Backup Key Generation', () => {
  test('should generate correct age recipient from test vector mnemonic', async () => {
    // Test vector
    const mnemonic = 'stay local melt rude evoke pause input kite area sphere mango quote'.split(' ')
    const expectedRecipient = 'age1f8ygkw8sqtucpj78lq4mund4gjsaq3zpfc9rmm4sngkhmccmgfpsuj7vzw'

    // Convert mnemonic to age recipient
    const actualRecipient = await wordsToAgeRecipient(mnemonic)

    // Test vector verification passed

    expect(actualRecipient).toBe(expectedRecipient)
  })

  test('should generate identity from test vector mnemonic', () => {
    const mnemonic = 'stay local melt rude evoke pause input kite area sphere mango quote'.split(' ')

    // This should not throw an error
    const identity = createIdentityFromWords(mnemonic)

    // Identity should start with AGE-SECRET-KEY-
    expect(identity).toMatch(/^AGE-SECRET-KEY-[A-Z0-9]+$/)
  })

  test('should have correct mnemonic word count', () => {
    const mnemonic = 'stay local melt rude evoke pause input kite area sphere mango quote'.split(' ')
    expect(mnemonic).toHaveLength(12)
  })
})