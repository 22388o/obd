package rpc

import (
	"log"
	"testing"
)

func TestClient_OmniListTransactions(t *testing.T) {
	client := NewClient()

	//hex := "02000000026fd85addeff9e017e1031b25c86e41aa37e82aa19897e1466b5e8bf9d6c8fb9000000000da00473044022076256e6c1dd01323c4299c6c49a13953e48ae713bdb10b46dc975ea459f136ce02201fed7f6b50a0d95b3ddd9f98fc59fe8b77478be48600b6a8afcc2703417c8535014830450221009abf41a3623e92481ce560b1fba19cfbd32fcfa924fbc426cd923853cccc379a022041f10fe7b24aa3c48e4043826bb014f30334b4ade255091af8e281c51f4313a30147522102746b20a865c3a152050fb57c47f6f652aa5f9067c2196d82f612fa5fecfbd1e021032f55cc908de7d95a5e587906c50deb9559ac621d889f3ee6be973de809c7e97b52aee80300006fd85addeff9e017e1031b25c86e41aa37e82aa19897e1466b5e8bf9d6c8fb9002000000db004830450221008ad1997f32b76f33de81c860ce01340e8a5550952117081ac57759f9de26a1dd022032299cb841c5a6cc1191a1b33456870a45a22f09d8d9f22e6a6e92fb86dc97b901483045022100bbb6a5e598b260c61f86d956f7930fd71f1e69aef10b36a38fdc216ccbbded7d022008c737a38043a8b236916fa5146bd0fec576cf2510263644af4b2e9c79a9054e0147522102746b20a865c3a152050fb57c47f6f652aa5f9067c2196d82f612fa5fecfbd1e021032f55cc908de7d95a5e587906c50deb9559ac621d889f3ee6be973de809c7e97b52aee8030000034a140000000000001976a914928f34815d1a8f54afe239ad68391fcddb505a6588ac0000000000000000166a146f6d6e690000000000000089000000001dcd650022020000000000001976a914928f34815d1a8f54afe239ad68391fcddb505a6588ac00000000"
	//result, err := client.SendRawTransaction(hex)
	//log.Println(result)
	//log.Println(err)

	//s, err := client.GetAddressInfo("2NAz6DPFP4PSN5tPZ1CA9KZi74NqjfqpL3A")
	//log.Println(s)
	//log.Println(err)
	hex := "02000000019b3dc9db592d9033aae2165361f50338ed746743249a5cf8c7f949e676a6c51100000000da00473044022055386e0a0fed62d664c1d0ff3719f1d6ee59d39e3d19b9f0bda866fc1c14bf6102205099baf32765ef72ba62e6836b5e64e5eec7eb0dc2b1b5e5ebecab24387553fc01483045022100e2fc49d56de99492337b90a70cd1ceaed443bbe4f67e061c9f840c5c5e59339b0220706fde685aea7d12b6cd67494e3c33b001ee84e7d68ca42b1dc0e05fd3cdface0147522103c384b8d9c65edea28ce205537bb58dc0096bc618e9e553455e1db1f36cc256422103ac744b776fc42bb700f601770a4ce19767c462391711ab0ee3340a1b0731cd2652aeffffffff033c1b00000000000017a9146f067afbadd31955a0d126fd255401721b984ee8870000000000000000166a146f6d6e6900000000000000890000000005f5e100220200000000000017a9146f067afbadd31955a0d126fd255401721b984ee88700000000"
	transaction, err := client.OmniDecodeTransaction(hex)
	log.Println(err)
	log.Println(transaction)

}