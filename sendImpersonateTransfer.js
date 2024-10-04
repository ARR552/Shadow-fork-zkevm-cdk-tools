const { ethers, network } = require("hardhat");

async function send() {
  const sequencer_address = "0x761d53b47334bEe6612c0Bd1467FB881435375B2";
  const addressTo = "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266";

  //  impersonating sequencer's account
  await network.provider.request({
    method: "hardhat_impersonateAccount",
    params: [sequencer_address],
  });

  //   make sequencer the signer
  const signer = await ethers.getSigner(sequencer_address);
  const accountBalance = await signer.provider.getBalance(signer.address);
  console.log(
    "sequencer account before transaction",
    accountBalance
  );

  //   create  transaction
  const tx = {
    from: sequencer_address,
    to: addressTo,
    value: "1",
  };

  let recieptTx = await signer.sendTransaction(tx);

  await recieptTx.wait();

  recieptTx = await signer.sendTransaction(tx);

  await recieptTx.wait();

  console.log(`Transaction successful with hash: ${recieptTx.hash}`);
  console.log(
    "sequencer account after transaction",
    await signer.provider.getBalance(signer.address)
  );
  console.log(
    "receiver account after transaction",
    await signer.provider.getBalance(addressTo)
  );
}

send()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });