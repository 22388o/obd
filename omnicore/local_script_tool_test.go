package omnicore

import (
	"github.com/omnilaboratory/obd/bean"
	"github.com/omnilaboratory/obd/tool"
	"log"
	"testing"
)

func TestCreateMultiSig(t *testing.T) {
	s, i, i2 := CreateMultiSigAddr("02c57b02d24356e1d31d34d2e3a09f7d68a4bdec6c0556595bb6391ce5d6d4fc66", "032dedba91b8ed7fb32dec1e2270bd451dee3521d1d9f53059a05830b4aa0d635b", tool.GetCoreNet())
	//s, i, i2 := CreateMultiSigAddr("02c4483151ede561fa04e465b47db1c0309af7f1afe753baedaac46a2d2e2a73c8", "032dedba91b8ed7fb32dec1e2270bd451dee3521d1d9f53059a05830b4aa0d635b", tool.GetCoreNet())
	log.Println(s)
	log.Println(i)
	log.Println(i2)
}

func TestVerifyOmniTxHex(t *testing.T) {
	hex1 := "020000000116947d93ddccf2e1680351906ae317bb1cd3432829a16489187dea1bacce73da0000000092004730440220313b30450e52b9ced1644bf2c1a94e326dda29538cc85c3a4aec22049a08b79702205557d0731df790d3e4c6283d80365312bbcb39c0a5367997c9734efd390b7733010047522102241c98bb5c64f34bd414f05e722924a6946569f817190bbe9752f1c0416b20dc21028b3c5073c29c243151a43889c92db19f31ccd8dd265059c81b7d4f75793dd10952aeffffffff033c1b00000000000017a9140c47da8faceace1c0cbaeacbd58909112663aca8870000000000000000166a146f6d6e69000000000000008900000000002d78a0220200000000000017a9140c47da8faceace1c0cbaeacbd58909112663aca88700000000"
	transaction := DecodeRawTransaction(hex1, tool.GetCoreNet())
	log.Println(transaction)
}

func TestSign(t *testing.T) {
	redeemhex := "0200000002ac5a7c14bf3a63944b50333e12263213fdf59eacfc1a60af27298324db736aea00000000930000483045022100c4acce9704328d6ffd85e1c4cd358b999c9590adff9db3ddf24a388f282cc8cf022024660b7321d7502e4ca71b989cda642d676714c400a6bbc4b5f2dd205f512d0c0147522103d2586577e2f4460b1c299f4f74b719982c1982f31e48ca1ef406347472fa611221038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52aeffffffffac5a7c14bf3a63944b50333e12263213fdf59eacfc1a60af27298324db736aea020000009200004730440220701fde05a432c78f541aaa4e10a714e79a60daa3d764d0830ca2195d7d85f325022053cf5f59dad9e3322df53990eea4a6fe4aa2455d63e72bbabe07cc3189ca69fe0147522103d2586577e2f4460b1c299f4f74b719982c1982f31e48ca1ef406347472fa611221038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52aeffffffff034a140000000000001976a914a2bebc3bbc138a248296ad96e6aaf71d83f69c3688ac0000000000000000166a146f6d6e69000000000000008900000000058b114022020000000000001976a914c18bb19ca8f23be298fd305f06f4e039cb10dca088ac00000000"
	privkey := "cRvLERMVjEND2XGi1YEgPjQT6KkshQadJjtmBkbUgcQvJ5ZXNY6P"
	transaction := DecodeRawTransaction(redeemhex, tool.GetCoreNet())
	log.Println("redeemhex 部分签", transaction)

	redeemhexA := "0200000002ac5a7c14bf3a63944b50333e12263213fdf59eacfc1a60af27298324db736aea00000000da00473044022052864a0e9a3ba7175506b6aaf21229fed03ca05c42846de050e53603c55ff37302204b3f484f91f3e4ac799a98afca351309921c857c38298e351ca5a0328b37291101483045022100c4acce9704328d6ffd85e1c4cd358b999c9590adff9db3ddf24a388f282cc8cf022024660b7321d7502e4ca71b989cda642d676714c400a6bbc4b5f2dd205f512d0c0147522103d2586577e2f4460b1c299f4f74b719982c1982f31e48ca1ef406347472fa611221038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52aeffffffffac5a7c14bf3a63944b50333e12263213fdf59eacfc1a60af27298324db736aea02000000d9004730440220410c2c64d9cf4a5b9ffd390a3f2b9cb493c34990da20d00b3dbbbec0195a4df202205dea90808f168b2f5dc61f4bb3e5293688cb5cdfa8b851e4500e615e749486a0014730440220701fde05a432c78f541aaa4e10a714e79a60daa3d764d0830ca2195d7d85f325022053cf5f59dad9e3322df53990eea4a6fe4aa2455d63e72bbabe07cc3189ca69fe0147522103d2586577e2f4460b1c299f4f74b719982c1982f31e48ca1ef406347472fa611221038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52aeffffffff034a140000000000001976a914a2bebc3bbc138a248296ad96e6aaf71d83f69c3688ac0000000000000000166a146f6d6e69000000000000008900000000058b114022020000000000001976a914c18bb19ca8f23be298fd305f06f4e039cb10dca088ac00000000"
	transaction = DecodeRawTransaction(redeemhexA, tool.GetCoreNet())
	log.Println("redeemhexA", transaction)

	inputs := []bean.RawTxInputItem{}
	item := bean.RawTxInputItem{}
	item.ScriptPubKey = "a914a1617398d1a34529bbd35eeb5c30a4ce20a73d2b87"
	redeemScript := "522103d2586577e2f4460b1c299f4f74b719982c1982f31e48ca1ef406347472fa611221038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52ae"
	item.RedeemScript = redeemScript
	inputs = append(inputs, item)
	item = bean.RawTxInputItem{}
	item.ScriptPubKey = "a914a1617398d1a34529bbd35eeb5c30a4ce20a73d2b87"
	redeemScript = "522103d2586577e2f4460b1c299f4f74b719982c1982f31e48ca1ef406347472fa611221038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52ae"
	item.RedeemScript = redeemScript
	inputs = append(inputs, item)

	sign, err := SignRawHex(inputs, redeemhex, privkey)
	log.Println(err)
	log.Println(sign)
	transaction = DecodeRawTransaction(sign, tool.GetCoreNet())
	log.Println("sign 完成", transaction)
}

func TestSign4(t *testing.T) {
	sourcehex := "02000000021733ee13399d1e21c7e7fdb54ca1592bacd93ad691174486af55e8e335f9edbc00000000d900473044022066c6c4061564a00b5bb4823ca47d6ccee35be9e18e6aff0cafe776b1835bf6b202201d92f1c318bf33c662b0f9fc51b438976f48d3ee0cbab9352aab153047329f810147304402203f9c4cbf91cb0686ef89a5d1a2ef0d6fac057156a9b0f2ef0eb0feb15737d55b0220303f8be69010ee078ec2680e0952b5eb0edb39600de2a2c11f339a0097c0808a0147522102ab22188dd37966ab8f56fc36559c02fe0c498e492c90fc20f740ec8f45aff30021038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52aeffffffff1733ee13399d1e21c7e7fdb54ca1592bacd93ad691174486af55e8e335f9edbc02000000da00473044022057b988b41809b514a15e1785ffddbd15a949f2a7c99695c65402cf01e2a8a92a022045a4e6226bc7ce7b58982179f380e57762373d5f4b9b9bda96c93b7b7e1b082801483045022100f1847c4b201063cfa12b07cf2255df862e08026cd5e935ad210922941431551e02200708f4a2328ecf91cdcdd9b249a1adf33b9ea9cda21f6382bc58ca88d7c1a4790147522102ab22188dd37966ab8f56fc36559c02fe0c498e492c90fc20f740ec8f45aff30021038097033cb34a88b8bfc052adbbfefa8e92c33b7635e30c7a79d90ff4917c6c0b52aeffffffff034a140000000000001976a914a2bebc3bbc138a248296ad96e6aaf71d83f69c3688ac0000000000000000166a146f6d6e6900000000000000890000000005a995c022020000000000001976a914c18bb19ca8f23be298fd305f06f4e039cb10dca088ac00000000"
	err := DecodeRawTransaction(sourcehex, tool.GetCoreNet())
	log.Println(err)
}
