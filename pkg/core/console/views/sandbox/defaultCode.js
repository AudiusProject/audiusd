// Connect to MetaMask and fetch a track from Audius
const { createWalletClient, custom } = window.viem;
console.log("🦊 Checking for MetaMask...");

// Use only MetaMask (ignore Phantom)
const providers = window.ethereum?.providers || [window.ethereum];
const metamask = providers.find(p => p?.isMetaMask);

if (!metamask) {
  alert('MetaMask not found. Disable Phantom or other wallets for now.');
  throw new Error('MetaMask required');
}

console.log("🦊 MetaMask found");

const audiusChain = {
  id: 1056801,
  name: 'Audius',
  nativeCurrency: { name: '-', symbol: '-', decimals: 18 },
  rpcUrls: {
    default: { http: ['https://node1.audiusd.devnet/core/erpc'] }
  }
}

const [address] = await window.ethereum.request({
  method: 'eth_requestAccounts'
})

const audiusWalletClient = createWalletClient({
  account: address,
  chain: audiusChain,
  transport: custom(metamask),
});

await audiusWalletClient.addChain({ chain: audiusChain })
await audiusWalletClient.switchChain({ id: audiusChain.id })

window.walletClient = audiusWalletClient;
window.walletAddress = address;

const sdk = window.audiusSdk({
  appName: 'MonacoApp',
  environment: 'staging',
  services: { audiusWalletClient }
});

console.log("✅ Audius SDK initialized");

if (!sdk) {
  throw new Error("❌ Audius SDK not initialized.");
}

const res = await sdk.users.updateProfile({
  userId: 'mEx6RYQ',
  metadata: {
    bio: 'up and coming artist from the Bronx',
  },
  onProgress: (progress) => console.log('Progress: ', progress),
})
console.log({ res })
