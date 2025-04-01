Rewards that are attested by core are validated against the embedded files found pkg/core/rewards directory.

An attestation can be obtained from core through the grpc GetRewardAttestation call and the accompanying rest variant, `/core/grpc/attest/reward`. To get an attestation you pass in two parameters to the request:

`data` - canonicalized json data in the shape of the claim schema, base64 encoded. This can be passed in the grpc message or as a param in the GET request.
`signature` - signed sha256 hash of the data.

Implementations of this can be found down below:

```
// golang
func generateCanonicalSignedClaim(t *testing.T, privKey *ecdsa.PrivateKey, claim rewards.RewardClaim) (string, string) {

raw, err := json.Marshal(claim)

require.NoError(t, err)

  

canonicalJSON, err := jsoncanonicalizer.Transform(raw)

require.NoError(t, err)

  

hash := sha256.Sum256(canonicalJSON)

sig, err := crypto.Sign(hash[:], privKey)

require.NoError(t, err)

  

return base64.StdEncoding.EncodeToString(canonicalJSON), hex.EncodeToString(sig)
}
```

```
# python
import json
import hashlib
import base64
import binascii
from eth_keys import keys
from eth_utils import decode_hex
import json_canonical

def generate_canonical_signed_claim(private_key_hex: str, claim: dict):
    # Serialize and canonicalize
    raw = json.dumps(claim, separators=(",", ":"), sort_keys=True).encode()
    canonical_json = json_canonical.canonicalize(claim)

    # Hash the canonical JSON
    hash_bytes = hashlib.sha256(canonical_json).digest()

    # Sign using eth_keys
    private_key = keys.PrivateKey(decode_hex(private_key_hex))
    signature = private_key.sign_msg_hash(hash_bytes)

    return base64.b64encode(canonical_json).decode(), signature.to_hex()

# Example usage
# claim = { "user": "alice", "amount": 100 }
# private_key_hex = "0x..."
# base64_json, hex_sig = generate_canonical_signed_claim(private_key_hex, claim)

```

```
// typescript
import { createHash } from 'crypto';
import { ec as EC } from 'elliptic';
import canonicalize from 'canonicalize';

const ec = new EC('secp256k1');

export function generateCanonicalSignedClaim(privKeyHex: string, claim: object): [string, string] {
  const canonicalJSON = canonicalize(claim);
  if (!canonicalJSON) throw new Error('Canonicalization failed');

  const hash = createHash('sha256').update(canonicalJSON).digest();

  const key = ec.keyFromPrivate(privKeyHex.replace(/^0x/, ''), 'hex');
  const signature = key.sign(hash);

  const derSignatureHex = signature.toDER('hex');

  return [
    Buffer.from(canonicalJSON).toString('base64'),
    derSignatureHex
  ];
}

// Example usage
// const [base64JSON, hexSig] = generateCanonicalSignedClaim('privkey...', { user: 'alice', amount: 100 });

```

Once the signed message and the data has been sent to core, core will validate that the structure and parameters are valid based on the relevant rewards for that environment. See `/pkg/core/rewards/{env}/rewards.json` for specifics. 

Core will validate the amount is correct, the shape of the json is canonicalized, and that the recovered signer is one of the relevant ones listed in the `rewards.json`.

Should these things pass core will return a similar structure. A `data` field with the original data plus the signature from the sender and a signature signed by it's validator key attesting that this is a valid reward. The caller can then acquire multiple of these attestations if required and use these attestations for rewards of their own need.