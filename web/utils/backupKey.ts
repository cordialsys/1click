import * as bip39 from 'bip39'
import { bech32 } from 'bech32'
import { identityToRecipient } from 'age-encryption'

export interface BackupKey {
  mnemonic: string[]
  ageRecipient: string
}

// Convert bits as done in Go bech32.ConvertBits
function convertBits(data: number[], fromBits: number, toBits: number, pad: boolean): number[] {
  let acc = 0
  let bits = 0
  const result: number[] = []
  const maxv = (1 << toBits) - 1
  const maxAcc = (1 << (fromBits + toBits - 1)) - 1

  for (const value of data) {
    if (value < 0 || value >> fromBits) {
      throw new Error('Invalid input value')
    }
    acc = ((acc << fromBits) | value) & maxAcc
    bits += fromBits
    while (bits >= toBits) {
      bits -= toBits
      result.push((acc >> bits) & maxv)
    }
  }

  if (pad) {
    if (bits) {
      result.push((acc << (toBits - bits)) & maxv)
    }
  } else if (bits >= fromBits || ((acc << (toBits - bits)) & maxv)) {
    throw new Error('Invalid padding')
  }

  return result
}

// Convert entropy to age identity string and then to recipient (like Go implementation)
function identityFromBytes(key: Uint8Array): string {
  // Pad to 32 bytes if needed (like Go implementation)
  let ed25519Key = new Uint8Array(32)
  if (key.length === 16) {
    ed25519Key.set(key, 16) // Copy to second half like Go: copy(ed255Key[16:], key)
  } else {
    ed25519Key.set(key)
  }

  // Convert to 5-bit groups for bech32 (like Go bech32.ConvertBits)
  const secret5Bit = convertBits(Array.from(ed25519Key), 8, 5, true)

  // Encode as AGE-SECRET-KEY (like Go bech32.Encode)
  const ageSecret = bech32.encode('age-secret-key-', secret5Bit).toUpperCase()

  return ageSecret
}

// Convert entropy to age recipient public key
async function entropyToAgeRecipient(entropy: Uint8Array): Promise<string> {
  const identity = identityFromBytes(entropy)
  return await identityToRecipient(identity)
}

// Reverse BIP39 mnemonic to entropy (like Go MnemonicToEntropy)
function mnemonicToEntropy(words: string[]): Uint8Array {
  if (words.length !== 12 && words.length !== 24) {
    throw new Error('Mnemonic must be 12 or 24 words')
  }

  const entropy = new Uint8Array(33)
  let cursor = 0
  let offset = 0
  let remainder = 0

  for (const word of words) {
    const wordList = bip39.wordlists.english
    const index = wordList.indexOf(word)
    if (index === -1) {
      throw new Error(`Invalid word: ${word}`)
    }

    remainder |= (index << (32 - 11)) >>> offset
    offset += 11

    while (offset >= 8) {
      entropy[cursor] = remainder >>> 24
      cursor += 1
      remainder <<= 8
      offset -= 8
    }
  }

  if (offset !== 0) {
    entropy[cursor] = remainder >>> 24
  }

  const entropyBytes = Math.floor(words.length / 3) * 4
  return entropy.slice(0, entropyBytes)
}

// Generate a backup key (equivalent to Go GenerateEncryptionKey)
export async function generateBackupKey(): Promise<BackupKey> {
  // Generate 16 bytes of entropy (128 bits = 12 words)
  const entropy = new Uint8Array(16)
  if (typeof window !== 'undefined' && window.crypto) {
    window.crypto.getRandomValues(entropy)
  } else {
    // Fallback for server-side or old browsers
    for (let i = 0; i < entropy.length; i++) {
      entropy[i] = Math.floor(Math.random() * 256)
    }
  }

  // Generate mnemonic from entropy
  const mnemonic = bip39.entropyToMnemonic(Buffer.from(entropy))
  const words = mnemonic.split(' ')

  // Generate age recipient
  const ageRecipient = await entropyToAgeRecipient(entropy)

  return {
    mnemonic: words,
    ageRecipient
  }
}

// Validate an age recipient (should start with 'age1')
export function validateAgeRecipient(recipient: string): boolean {
  return recipient.startsWith('age1') && recipient.length >= 60
}

// Convert mnemonic words back to age recipient (for verification)
export async function wordsToAgeRecipient(words: string[]): Promise<string> {
  const entropy = mnemonicToEntropy(words)
  return await entropyToAgeRecipient(entropy)
}

// Create identity from mnemonic words (like Go NewEncryptionKey)
export function createIdentityFromWords(words: string[]): string {
  const entropy = mnemonicToEntropy(words)
  return identityFromBytes(entropy)
}