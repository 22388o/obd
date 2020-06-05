package service

import (
	"encoding/json"
	"errors"
	"github.com/asdine/storm/q"
	"github.com/omnilaboratory/obd/bean"
	"github.com/omnilaboratory/obd/dao"
	"github.com/omnilaboratory/obd/tool"
	"github.com/tidwall/gjson"
	"log"
	"strconv"
	"sync"
	"time"
)

//close htlc or close channel
type htlcCloseTxManager struct {
	operationFlag sync.Mutex
}

// htlc 关闭当前htlc交易
var HtlcCloseTxService htlcCloseTxManager

// -49 request close htlc
func (service *htlcCloseTxManager) RequestCloseHtlc(msg bean.RequestMessage, user bean.User) (retData map[string]interface{}, err error) {
	if tool.CheckIsString(&msg.Data) == false {
		return nil, errors.New("empty json data")
	}

	reqData := &bean.HtlcRequestCloseCurrTx{}
	err = json.Unmarshal([]byte(msg.Data), reqData)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// region check data
	if tool.CheckIsString(&reqData.ChannelId) == false {
		err = errors.New("wrong channel Id")
		log.Println(err)
		return nil, err
	}

	if tool.CheckIsString(&reqData.CurrRsmcTempAddressPubKey) == false {
		err = errors.New("empty CurrRsmcTempAddressPubKey")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.CurrRsmcTempAddressPrivateKey) == false {
		err = errors.New("empty CurrRsmcTempAddressPrivateKey")
		log.Println(err)
		return nil, err
	}

	_, err = tool.GetPubKeyFromWifAndCheck(reqData.CurrRsmcTempAddressPrivateKey, reqData.CurrRsmcTempAddressPubKey)
	if err != nil {
		return nil, errors.New("CurrRsmcTempAddressPrivateKey is wrong")
	}

	if tool.CheckIsString(&reqData.LastHtlcTempAddressForHtnxPrivateKey) == false {
		err = errors.New("empty LastHtlcTempAddressForHtnxPrivateKey")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.LastHtlcTempAddressPrivateKey) == false {
		err = errors.New("empty LastHtlcTempAddressPrivateKey")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.LastRsmcTempAddressPrivateKey) == false {
		err = errors.New("empty LastRsmcTempAddressPrivateKey")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.ChannelAddressPrivateKey) == false {
		err = errors.New("empty ChannelAddressPrivateKey")
		log.Println(err)
		return nil, err
	}
	// endregion

	tx, err := user.Db.Begin(true)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer tx.Rollback()

	channelInfo := &dao.ChannelInfo{}
	err = tx.Select(
		q.Eq("ChannelId", reqData.ChannelId),
		q.Or(
			q.Eq("PeerIdA", user.PeerId),
			q.Eq("PeerIdB", user.PeerId))).
		First(channelInfo)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if channelInfo.CurrState != dao.ChannelState_HtlcTx {
		return nil, errors.New("wrong channel state")
	}

	requesterPubKey := channelInfo.PubKeyA
	targetUser := channelInfo.PeerIdB
	if user.PeerId == channelInfo.PeerIdB {
		requesterPubKey = channelInfo.PubKeyB
		targetUser = channelInfo.PeerIdA
	}
	if msg.RecipientUserPeerId != targetUser {
		return nil, errors.New("RecipientUserPeerId is wrong")
	}
	if P2PLocalPeerId == msg.RecipientNodePeerId {
		if err := FindUserIsOnline(targetUser); err != nil {
			return nil, err
		}
	}
	_, err = tool.GetPubKeyFromWifAndCheck(reqData.ChannelAddressPrivateKey, requesterPubKey)
	if err != nil {
		return nil, errors.New("ChannelAddressPrivateKey is wrong")
	}

	latestCommitmentTxInfo, err := getLatestCommitmentTxUseDbTx(tx, reqData.ChannelId, user.PeerId)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if latestCommitmentTxInfo.CurrState != dao.TxInfoState_Htlc_GetR &&
		latestCommitmentTxInfo.CurrState != dao.TxInfoState_Create {
		return nil, errors.New("latestCommitmentTxInfo currState is not in TxInfoState_Htlc_GetR or TxInfoState_Create state")
	}

	ht1aOrHe1b := dao.HTLCTimeoutTxForAAndExecutionForB{}
	if latestCommitmentTxInfo.CurrState == dao.TxInfoState_Htlc_GetR {
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastRsmcTempAddressPrivateKey, latestCommitmentTxInfo.RSMCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastRsmcTempAddressPrivateKey is wrong")
		}
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastHtlcTempAddressPrivateKey, latestCommitmentTxInfo.HTLCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastHtlcTempAddressPrivateKey is wrong")
		}

		err = tx.Select(
			q.Eq("ChannelId", channelInfo.ChannelId),
			q.Eq("CommitmentTxId", latestCommitmentTxInfo.Id),
			q.Eq("Owner", user.PeerId)).
			First(&ht1aOrHe1b)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	} else {
		lastCommitmentTxInfo := &dao.CommitmentTransaction{}
		err = tx.One("Id", latestCommitmentTxInfo.LastCommitmentTxId, lastCommitmentTxInfo)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastRsmcTempAddressPrivateKey, lastCommitmentTxInfo.RSMCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastRsmcTempAddressPrivateKey is wrong")
		}
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastHtlcTempAddressPrivateKey, lastCommitmentTxInfo.HTLCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastHtlcTempAddressPrivateKey is wrong")
		}
		err = tx.Select(
			q.Eq("ChannelId", channelInfo.ChannelId),
			q.Eq("CommitmentTxId", lastCommitmentTxInfo.Id),
			q.Eq("Owner", user.PeerId)).
			First(&ht1aOrHe1b)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}

	_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastHtlcTempAddressForHtnxPrivateKey, ht1aOrHe1b.RSMCTempAddressPubKey)
	if err != nil {
		return nil, errors.New("LastHtlcTempAddressForHtnxPrivateKey is wrong")
	}
	tempAddrPrivateKeyMap[reqData.CurrRsmcTempAddressPubKey] = reqData.CurrRsmcTempAddressPrivateKey
	tempAddrPrivateKeyMap[requesterPubKey] = reqData.ChannelAddressPrivateKey

	retData = make(map[string]interface{})
	retData["channelId"] = channelInfo.ChannelId
	retData["lastRsmcTempAddressPrivateKey"] = reqData.LastRsmcTempAddressPrivateKey
	retData["lastHtlcTempAddressPrivateKey"] = reqData.LastHtlcTempAddressPrivateKey
	retData["lastHtlcTempAddressForHtnxPrivateKey"] = reqData.LastHtlcTempAddressForHtnxPrivateKey
	retData["currRsmcTempAddressPubKey"] = reqData.CurrRsmcTempAddressPubKey

	//如果是第一次请求，前面没有请求失败
	if latestCommitmentTxInfo.CurrState == dao.TxInfoState_Htlc_GetR {
		//创建c2a omni的交易不能一个输入，多个输出，所以就是两个交易
		reqTempData := &bean.CommitmentTx{}
		reqTempData.CurrTempAddressPubKey = reqData.CurrRsmcTempAddressPubKey
		reqTempData.ChannelAddressPrivateKey = reqData.ChannelAddressPrivateKey
		reqTempData.Amount = 0
		newCommitmentTxInfo, err := createCommitmentTxHex(tx, true, reqTempData, channelInfo, latestCommitmentTxInfo, user)
		if err != nil {
			return nil, err
		}
		retData["rsmcHex"] = newCommitmentTxInfo.RSMCTxHex
		retData["toOtherHex"] = newCommitmentTxInfo.ToCounterpartyTxHex
		retData["commitmentHash"] = newCommitmentTxInfo.CurrHash
	} else {
		retData["rsmcHex"] = latestCommitmentTxInfo.RSMCTxHex
		retData["toOtherHex"] = latestCommitmentTxInfo.ToCounterpartyTxHex
		retData["commitmentHash"] = latestCommitmentTxInfo.CurrHash
	}
	_ = tx.Commit()
	return retData, nil
}

// -49 接收方节点的信息缓存处理
func (service *htlcCloseTxManager) BeforeBobSignCloseHtlcAtBobSide(data string, user *bean.User) (retData map[string]interface{}, err error) {
	jsonObj := gjson.Parse(data)
	retData = make(map[string]interface{})
	retData["channelId"] = jsonObj.Get("channelId").String()
	retData["rsmcHex"] = jsonObj.Get("rsmcHex").String()
	retData["toOtherHex"] = jsonObj.Get("toOtherHex").String()

	tx, err := user.Db.Begin(true)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer tx.Rollback()

	channelInfo := &dao.ChannelInfo{}
	err = tx.Select(
		q.Eq("ChannelId", jsonObj.Get("channelId").String()),
		q.Or(
			q.Eq("PeerIdA", user.PeerId),
			q.Eq("PeerIdB", user.PeerId))).
		First(channelInfo)
	if err != nil {
		return nil, err
	}
	if channelInfo == nil {
		return nil, errors.New("not found channelInfo at targetSide")
	}

	senderPeerId := channelInfo.PeerIdA
	if user.PeerId == channelInfo.PeerIdA {
		senderPeerId = channelInfo.PeerIdB
	}
	messageHash := MessageService.saveMsgUseTx(tx, senderPeerId, user.PeerId, data)
	retData["msgHash"] = messageHash
	_ = tx.Commit()

	return retData, nil
}

// -50 sign close htlc
func (service *htlcCloseTxManager) CloseHTLCSigned(msg bean.RequestMessage, user bean.User) (retData map[string]interface{}, err error) {
	if tool.CheckIsString(&msg.Data) == false {
		return nil, errors.New("empty json data")
	}

	reqData := &bean.HtlcSignCloseCurrTx{}
	err = json.Unmarshal([]byte(msg.Data), reqData)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// region check data
	if tool.CheckIsString(&reqData.RequestHash) == false {
		err = errors.New("empty request_hash")
		log.Println(err)
		return nil, err
	}

	tx, err := user.Db.Begin(true)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer tx.Rollback()

	message, err := MessageService.getMsgUseTx(tx, reqData.RequestHash)
	if err != nil {
		return nil, errors.New("wrong request_hash")
	}
	if message.Receiver != user.PeerId {
		return nil, errors.New("you are not the operator")
	}
	aliceDataJson := gjson.Parse(message.Data)

	if tool.CheckIsString(&reqData.ChannelAddressPrivateKey) == false {
		err = errors.New("empty channel_address_private_key")
		log.Println(err)
		return nil, err
	}

	if tool.CheckIsString(&reqData.CurrRsmcTempAddressPubKey) == false {
		err = errors.New("empty curr_rsmc_temp_address_pub_key")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.CurrRsmcTempAddressPrivateKey) == false {
		err = errors.New("empty curr_rsmc_temp_address_private_key")
		log.Println(err)
		return nil, err
	}
	_, err = tool.GetPubKeyFromWifAndCheck(reqData.CurrRsmcTempAddressPrivateKey, reqData.CurrRsmcTempAddressPubKey)
	if err != nil {
		return nil, errors.New("CurrRsmcTempAddressPrivateKey is wrong")
	}
	tempAddrPrivateKeyMap[reqData.CurrRsmcTempAddressPubKey] = reqData.CurrRsmcTempAddressPrivateKey

	if tool.CheckIsString(&reqData.LastHtlcTempAddressForHtnxPrivateKey) == false {
		err = errors.New("empty last_htlc_temp_address_for_htnx_private_key")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.LastHtlcTempAddressPrivateKey) == false {
		err = errors.New("empty last_htlc_temp_address_private_key")
		log.Println(err)
		return nil, err
	}
	if tool.CheckIsString(&reqData.LastRsmcTempAddressPrivateKey) == false {
		err = errors.New("empty last_rsmc_temp_address_private_key")
		log.Println(err)
		return nil, err
	}
	// endregion

	channelInfo := &dao.ChannelInfo{}
	err = tx.Select(
		q.Eq("ChannelId", aliceDataJson.Get("channelId").String()),
		q.Or(
			q.Eq("PeerIdA", user.PeerId),
			q.Eq("PeerIdB", user.PeerId))).
		First(channelInfo)
	if err != nil {
		return nil, err
	}

	signerPubKey := channelInfo.PubKeyB
	senderPeerId := channelInfo.PeerIdA
	if user.PeerId == channelInfo.PeerIdA {
		senderPeerId = channelInfo.PeerIdB
		signerPubKey = channelInfo.PubKeyA
	}
	if senderPeerId != msg.RecipientUserPeerId {
		return nil, errors.New("wrong RecipientUserPeerId")
	}

	if P2PLocalPeerId == msg.RecipientNodePeerId {
		err = FindUserIsOnline(senderPeerId)
		if err != nil {
			return nil, err
		}
	}

	_, err = tool.GetPubKeyFromWifAndCheck(reqData.ChannelAddressPrivateKey, signerPubKey)
	if err != nil {
		return nil, errors.New("LastRsmcTempAddressPrivateKey is wrong")
	}
	tempAddrPrivateKeyMap[signerPubKey] = reqData.ChannelAddressPrivateKey

	latestCommitmentTxInfo, _ := getLatestCommitmentTxUseDbTx(tx, channelInfo.ChannelId, user.PeerId)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if latestCommitmentTxInfo.CurrState != dao.TxInfoState_Htlc_GetR && latestCommitmentTxInfo.CurrState != dao.TxInfoState_Create {
		return nil, errors.New("wrong commitment tx state " + strconv.Itoa(int(latestCommitmentTxInfo.CurrState)))
	}

	he1b := dao.HTLCTimeoutTxForAAndExecutionForB{}
	if latestCommitmentTxInfo.TxType == dao.CommitmentTransactionType_Htlc {
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastRsmcTempAddressPrivateKey, latestCommitmentTxInfo.RSMCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastRsmcTempAddressPrivateKey is wrong")
		}
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastHtlcTempAddressPrivateKey, latestCommitmentTxInfo.HTLCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastHtlcTempAddressPrivateKey is wrong")
		}
		err = tx.Select(
			q.Eq("ChannelId", channelInfo.ChannelId),
			q.Eq("CommitmentTxId", latestCommitmentTxInfo.Id),
			q.Eq("Owner", user.PeerId)).
			First(&he1b)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	} else {
		lastCommitTxInfo := dao.CommitmentTransaction{}
		err = tx.One("Id", latestCommitmentTxInfo.LastCommitmentTxId, &lastCommitTxInfo)
		if err != nil {
			return nil, err
		}
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastRsmcTempAddressPrivateKey, lastCommitTxInfo.RSMCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastRsmcTempAddressPrivateKey is wrong")
		}
		_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastHtlcTempAddressPrivateKey, lastCommitTxInfo.HTLCTempAddressPubKey)
		if err != nil {
			return nil, errors.New("LastHtlcTempAddressPrivateKey is wrong")
		}

		err = tx.Select(
			q.Eq("ChannelId", channelInfo.ChannelId),
			q.Eq("CommitmentTxId", lastCommitTxInfo.Id),
			q.Eq("Owner", user.PeerId)).
			First(&he1b)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}
	_, err = tool.GetPubKeyFromWifAndCheck(reqData.LastHtlcTempAddressForHtnxPrivateKey, he1b.RSMCTempAddressPubKey)
	if err != nil {
		return nil, errors.New("LastHtlcTempAddressForHtnxPrivateKey is wrong")
	}

	retData = make(map[string]interface{})
	retData["channelId"] = channelInfo.ChannelId
	retData["lastRsmcTempAddressPrivateKey"] = reqData.LastRsmcTempAddressPrivateKey
	retData["lastHtlcTempAddressPrivateKey"] = reqData.LastHtlcTempAddressPrivateKey
	retData["lastHtlcTempAddressForHtnxPrivateKey"] = reqData.LastHtlcTempAddressForHtnxPrivateKey
	retData["currRsmcTempAddressPubKey"] = reqData.CurrRsmcTempAddressPubKey

	// get the funding transaction
	fundingTransaction := getFundingTransactionByChannelId(tx, channelInfo.ChannelId, user.PeerId)
	if fundingTransaction == nil {
		return nil, errors.New("not found fundingTransaction")
	}

	// region 签名requester的承诺交易
	// 签名对方传过来的rsmcHex
	aliceRsmcTxId, signedRsmcHex, err := rpcClient.BtcSignRawTransaction(aliceDataJson.Get("rsmcHex").String(), reqData.ChannelAddressPrivateKey)
	if err != nil {
		return nil, errors.New("fail to sign rsmc hex ")
	}
	testResult, err := rpcClient.TestMemPoolAccept(signedRsmcHex)
	if err != nil {
		return nil, err
	}
	if gjson.Parse(testResult).Array()[0].Get("allowed").Bool() == false {
		return nil, errors.New(gjson.Parse(testResult).Array()[0].Get("reject-reason").String())
	}

	//  根据alice的临时地址+bob的通道address,获取alice2+bob的多签地址，并得到AliceSignedRsmcHex签名后的交易的input，为创建alice的RD和bob的BR做准备
	aliceMultiAddr, err := rpcClient.CreateMultiSig(2, []string{aliceDataJson.Get("currRsmcTempAddressPubKey").String(), signerPubKey})
	if err != nil {
		return nil, err
	}
	aliceRsmcMultiAddress := gjson.Get(aliceMultiAddr, "address").String()
	aliceRsmcRedeemScript := gjson.Get(aliceMultiAddr, "redeemScript").String()
	tempJson, err := rpcClient.GetAddressInfo(aliceRsmcMultiAddress)
	if err != nil {
		return nil, err
	}
	aliceRsmcMultiAddressScriptPubKey := gjson.Get(tempJson, "scriptPubKey").String()

	aliceRsmcOutputs, err := getInputsForNextTxByParseTxHashVout(signedRsmcHex, aliceRsmcMultiAddress, aliceRsmcMultiAddressScriptPubKey, aliceRsmcRedeemScript)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	retData["signedRsmcHex"] = signedRsmcHex

	//  签名对方传过来的toOtherHex
	_, signedToOtherHex, err := rpcClient.BtcSignRawTransaction(aliceDataJson.Get("toOtherHex").String(), reqData.ChannelAddressPrivateKey)
	if err != nil {
		return nil, errors.New("fail to sign toOther hex ")
	}
	testResult, err = rpcClient.TestMemPoolAccept(signedToOtherHex)
	if err != nil {
		return nil, err
	}
	if gjson.Parse(testResult).Array()[0].Get("allowed").Bool() == false {
		return nil, errors.New(gjson.Parse(testResult).Array()[0].Get("reject-reason").String())
	}
	retData["signedToOtherHex"] = signedToOtherHex
	//endregion

	//第一次签名，不是失败后的重试
	var amountToCounterparty = 0.0
	if latestCommitmentTxInfo.TxType == dao.CommitmentTransactionType_Htlc {
		//region 3、根据对方传过来的上一个交易的临时rsmc私钥，签名最近的BR交易，保证对方确实放弃了上一个承诺交易
		err = signLastBR(tx, dao.BRType_Rmsc, *channelInfo, user.PeerId, aliceDataJson.Get("lastRsmcTempAddressPrivateKey").String(), latestCommitmentTxInfo.Id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		err = signLastBR(tx, dao.BRType_Htlc, *channelInfo, user.PeerId, aliceDataJson.Get("lastHtlcTempAddressPrivateKey").String(), latestCommitmentTxInfo.Id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		err = signLastBR(tx, dao.BRType_Ht1a, *channelInfo, user.PeerId, aliceDataJson.Get("lastHtlcTempAddressForHtnxPrivateKey").String(), latestCommitmentTxInfo.Id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		//endregion

		//region 4、创建C2b
		commitmentTxRequest := &bean.CommitmentTx{}
		commitmentTxRequest.ChannelId = channelInfo.ChannelId
		commitmentTxRequest.Amount = 0
		commitmentTxRequest.ChannelAddressPrivateKey = reqData.ChannelAddressPrivateKey
		commitmentTxRequest.CurrTempAddressPubKey = reqData.CurrRsmcTempAddressPubKey
		commitmentTxRequest.CurrTempAddressPrivateKey = reqData.CurrRsmcTempAddressPrivateKey
		newCommitmentTxInfo, err := createCommitmentTxHex(tx, false, commitmentTxRequest, channelInfo, latestCommitmentTxInfo, user)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		amountToCounterparty = newCommitmentTxInfo.AmountToCounterparty

		retData["rsmcHex"] = newCommitmentTxInfo.RSMCTxHex
		retData["toOtherHex"] = newCommitmentTxInfo.ToCounterpartyTxHex
		//endregion

		// region 5、根据alice的Rsmc，创建对应的BR,为下一个交易做准备，create BR2b tx  for bob
		var myAddress = channelInfo.AddressB
		if user.PeerId == channelInfo.PeerIdA {
			myAddress = channelInfo.AddressA
		}
		senderCommitmentTx := &dao.CommitmentTransaction{}
		senderCommitmentTx.Id = newCommitmentTxInfo.Id
		senderCommitmentTx.PropertyId = fundingTransaction.PropertyId
		senderCommitmentTx.RSMCTempAddressPubKey = aliceDataJson.Get("currTempAddressPubKey").String()
		senderCommitmentTx.RSMCMultiAddress = aliceRsmcMultiAddress
		senderCommitmentTx.RSMCRedeemScript = aliceRsmcRedeemScript
		senderCommitmentTx.RSMCMultiAddressScriptPubKey = aliceRsmcMultiAddressScriptPubKey
		senderCommitmentTx.RSMCTxHex = signedRsmcHex
		senderCommitmentTx.RSMCTxid = aliceRsmcTxId
		senderCommitmentTx.AmountToRSMC = newCommitmentTxInfo.AmountToCounterparty
		err = createCurrCommitmentTxBR(tx, dao.BRType_Rmsc, channelInfo, senderCommitmentTx, aliceRsmcOutputs, myAddress, reqData.ChannelAddressPrivateKey, user)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		//endregion
	} else {
		retData["rsmcHex"] = latestCommitmentTxInfo.RSMCTxHex
		retData["toOtherHex"] = latestCommitmentTxInfo.ToCounterpartyTxHex
		amountToCounterparty = latestCommitmentTxInfo.AmountToCounterparty
	}

	//region 6、根据签名后的AliceRsmc创建alice的RD create RD tx for alice
	outputAddress := channelInfo.AddressA
	if user.PeerId == channelInfo.PeerIdA {
		outputAddress = channelInfo.AddressB
	}
	_, senderRdhex, err := rpcClient.OmniCreateAndSignRawTransactionUseUnsendInput(
		aliceRsmcMultiAddress,
		[]string{
			reqData.ChannelAddressPrivateKey,
		},
		aliceRsmcOutputs,
		outputAddress,
		channelInfo.FundingAddress,
		channelInfo.PropertyId,
		amountToCounterparty,
		0,
		1000,
		&aliceRsmcRedeemScript)
	if err != nil {
		log.Println(err)
		return nil, errors.New("fail to create rd")
	}
	retData["senderRdHex"] = senderRdhex
	retData["commitmentTxHash"] = aliceDataJson.Get("commitmentHash").Str
	//endregion create RD tx for alice

	_ = MessageService.updateMsgStateUseTx(tx, message)
	err = tx.Commit()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return retData, nil
}

// -51 alice根据得到的bob的临时私钥签名
func (service *htlcCloseTxManager) AfterBobCloseHTLCSigned_AtAliceSide(data string, user *bean.User) (retData map[string]interface{}, needNoticeAlice bool, err error) {
	jsonObj := gjson.Parse(data)

	//region 检测传入数据
	var channelId = jsonObj.Get("channelId").String()
	if tool.CheckIsString(&channelId) == false {
		err = errors.New("wrong channelId")
		log.Println(err)
		return nil, false, err
	}
	var commitmentTxHash = jsonObj.Get("commitmentTxHash").String()
	if tool.CheckIsString(&commitmentTxHash) == false {
		err = errors.New("wrong commitmentHash")
		log.Println(err)
		return nil, false, err
	}

	var signedRsmcHex = jsonObj.Get("signedRsmcHex").String()
	if tool.CheckIsString(&signedRsmcHex) == false {
		err = errors.New("wrong signedRsmcHex")
		log.Println(err)
		return nil, false, err
	}
	_, err = rpcClient.TestMemPoolAccept(signedRsmcHex)
	if err != nil {
		err = errors.New("wrong signedRsmcHex")
		log.Println(err)
		return nil, false, err
	}

	var signedToOtherHex = jsonObj.Get("signedToOtherHex").String()
	if tool.CheckIsString(&signedToOtherHex) == false {
		err = errors.New("wrong signedToOtherHex")
		log.Println(err)
		return nil, false, err
	}
	_, err = rpcClient.TestMemPoolAccept(signedToOtherHex)
	if err != nil {
		err = errors.New("wrong signedToOtherHex")
		log.Println(err)
		return nil, false, err
	}

	var aliceRdHex = jsonObj.Get("senderRdHex").String()
	if tool.CheckIsString(&aliceRdHex) == false {
		err = errors.New("wrong senderRdHex")
		log.Println(err)
		return nil, false, err
	}
	_, err = rpcClient.TestMemPoolAccept(aliceRdHex)
	if err != nil {
		err = errors.New("wrong senderRdHex")
		log.Println(err)
		return nil, false, err
	}

	var bobRsmcHex = jsonObj.Get("rsmcHex").String()
	if tool.CheckIsString(&bobRsmcHex) == false {
		err = errors.New("wrong rsmcHex")
		log.Println(err)
		return nil, false, err
	}
	_, err = rpcClient.TestMemPoolAccept(bobRsmcHex)
	if err != nil {
		err = errors.New("wrong rsmcHex")
		log.Println(err)
		return nil, false, err
	}

	var bobCurrTempAddressPubKey = jsonObj.Get("currRsmcTempAddressPubKey").String()
	if tool.CheckIsString(&bobCurrTempAddressPubKey) == false {
		err = errors.New("wrong currTempAddressPubKey")
		log.Println(err)
		return nil, false, err
	}
	var bobToOtherHex = jsonObj.Get("toOtherHex").String()
	if tool.CheckIsString(&bobToOtherHex) == false {
		err = errors.New("wrong toOtherHex")
		log.Println(err)
		return nil, false, err
	}
	//endregion

	tx, err := user.Db.Begin(true)
	if err != nil {
		log.Println(err)
		return nil, true, err
	}
	defer tx.Rollback()

	aliceData := make(map[string]interface{})
	bobData := make(map[string]interface{})

	channelInfo := &dao.ChannelInfo{}
	err = tx.Select(
		q.Eq("ChannelId", jsonObj.Get("channelId").String()),
		q.Or(
			q.Eq("PeerIdA", user.PeerId),
			q.Eq("PeerIdB", user.PeerId))).
		First(channelInfo)
	if err != nil {
		return nil, true, errors.New("not found channelInfo at targetSide")
	}

	fundingTransaction := getFundingTransactionByChannelId(tx, channelId, user.PeerId)
	if fundingTransaction == nil {
		return nil, true, errors.New("not found fundingTransaction at targetSide")
	}

	latestCcommitmentTxInfo, err := getLatestCommitmentTxUseDbTx(tx, channelId, user.PeerId)
	if err != nil {
		err = errors.New("fail to find sender's commitmentTxInfo")
		log.Println(err)
		return nil, true, err
	}

	if latestCcommitmentTxInfo.CurrHash != commitmentTxHash {
		err = errors.New("wrong request hash")
		log.Println(err)
		return nil, false, err
	}

	if latestCcommitmentTxInfo.CurrState != dao.TxInfoState_Create {
		err = errors.New("wrong commitmentTxInfo state " + strconv.Itoa(int(latestCcommitmentTxInfo.CurrState)))
		log.Println(err)
		return nil, false, err
	}

	var myChannelPubKey = channelInfo.PubKeyA
	var myChannelAddress = channelInfo.AddressA
	var partnerChannelAddress = channelInfo.AddressB
	if user.PeerId == channelInfo.PeerIdB {
		myChannelAddress = channelInfo.AddressB
		myChannelPubKey = channelInfo.PubKeyB
		partnerChannelAddress = channelInfo.AddressA
	}
	var myChannelPrivateKey = tempAddrPrivateKeyMap[myChannelPubKey]

	//region 根据对方传过来的上一个交易的临时rsmc私钥，签名上一次的BR交易，保证对方确实放弃了上一个承诺交易
	var bobLastRsmcTempAddressPrivateKey = jsonObj.Get("lastRsmcTempAddressPrivateKey").String()
	var bobLastHtlcTempAddressPrivateKey = jsonObj.Get("lastHtlcTempAddressPrivateKey").String()
	var bobLastHtlcTempAddressForHtnxPrivateKey = jsonObj.Get("lastHtlcTempAddressForHtnxPrivateKey").String()
	err = signLastBR(tx, dao.BRType_Rmsc, *channelInfo, user.PeerId, bobLastRsmcTempAddressPrivateKey, latestCcommitmentTxInfo.LastCommitmentTxId)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}
	err = signLastBR(tx, dao.BRType_Htlc, *channelInfo, user.PeerId, bobLastHtlcTempAddressPrivateKey, latestCcommitmentTxInfo.LastCommitmentTxId)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}
	err = signLastBR(tx, dao.BRType_HE1b, *channelInfo, user.PeerId, bobLastHtlcTempAddressForHtnxPrivateKey, latestCcommitmentTxInfo.LastCommitmentTxId)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}
	//endregion

	// region 对自己的RD 二次签名
	err = signRdTx(tx, channelInfo, signedRsmcHex, aliceRdHex, latestCcommitmentTxInfo, myChannelAddress, user)
	if err != nil {
		return nil, true, err
	}
	// endregion

	//更新alice的当前承诺交易
	latestCcommitmentTxInfo.SignAt = time.Now()
	latestCcommitmentTxInfo.CurrState = dao.TxInfoState_CreateAndSign
	latestCcommitmentTxInfo.RSMCTxHex = signedRsmcHex
	latestCcommitmentTxInfo.ToCounterpartyTxHex = signedToOtherHex
	bytes, err := json.Marshal(latestCcommitmentTxInfo)
	msgHash := tool.SignMsgWithSha256(bytes)
	latestCcommitmentTxInfo.CurrHash = msgHash
	_ = tx.Update(latestCcommitmentTxInfo)

	lastCommitmentTxInfo := dao.CommitmentTransaction{}
	err = tx.One("Id", latestCcommitmentTxInfo.LastCommitmentTxId, &lastCommitmentTxInfo)
	if err == nil {
		lastCommitmentTxInfo.CurrState = dao.TxInfoState_Abord
		_ = tx.Update(lastCommitmentTxInfo)
	}

	aliceData["latestCcommitmentTxInfo"] = latestCcommitmentTxInfo
	//处理对方的数据
	//签名对方传过来的rsmcHex
	bobRsmcTxid, bobSignedRsmcHex, err := rpcClient.BtcSignRawTransaction(bobRsmcHex, myChannelPrivateKey)
	if err != nil {
		return nil, false, errors.New("fail to sign rsmc hex ")
	}
	testResult, err := rpcClient.TestMemPoolAccept(bobSignedRsmcHex)
	if err != nil {
		return nil, false, err
	}
	if gjson.Parse(testResult).Array()[0].Get("allowed").Bool() == false {
		return nil, false, errors.New(gjson.Parse(testResult).Array()[0].Get("reject-reason").String())
	}
	err = checkBobRemcData(bobSignedRsmcHex, latestCcommitmentTxInfo)
	if err != nil {
		return nil, false, err
	}
	bobData["signedRsmcHex"] = bobSignedRsmcHex

	//region create RD tx for bob
	bobMultiAddr, err := rpcClient.CreateMultiSig(2, []string{bobCurrTempAddressPubKey, myChannelPubKey})
	if err != nil {
		return nil, false, err
	}
	bobRsmcMultiAddress := gjson.Get(bobMultiAddr, "address").String()
	bobRsmcRedeemScript := gjson.Get(bobMultiAddr, "redeemScript").String()
	addressJson, err := rpcClient.GetAddressInfo(bobRsmcMultiAddress)
	if err != nil {
		return nil, false, err
	}
	bobRsmcMultiAddressScriptPubKey := gjson.Get(addressJson, "scriptPubKey").String()

	inputs, err := getInputsForNextTxByParseTxHashVout(bobSignedRsmcHex, bobRsmcMultiAddress, bobRsmcMultiAddressScriptPubKey, bobRsmcRedeemScript)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}

	_, bobRdhex, err := rpcClient.OmniCreateAndSignRawTransactionUseUnsendInput(
		bobRsmcMultiAddress,
		[]string{
			myChannelPrivateKey,
		},
		inputs,
		partnerChannelAddress,
		channelInfo.FundingAddress,
		channelInfo.PropertyId,
		latestCcommitmentTxInfo.AmountToCounterparty,
		0,
		1000,
		&bobRsmcRedeemScript)
	if err != nil {
		log.Println(err)
		return nil, false, errors.New("fail to create rd")
	}
	bobData["rdHex"] = bobRdhex
	//endregion create RD tx for alice

	//region 根据对对方的Rsmc签名，生成惩罚对方，自己获益BR
	bobCommitmentTx := &dao.CommitmentTransaction{}
	bobCommitmentTx.Id = latestCcommitmentTxInfo.Id
	bobCommitmentTx.PropertyId = channelInfo.PropertyId
	bobCommitmentTx.RSMCTempAddressPubKey = bobCurrTempAddressPubKey
	bobCommitmentTx.RSMCMultiAddress = bobRsmcMultiAddress
	bobCommitmentTx.RSMCRedeemScript = bobRsmcRedeemScript
	bobCommitmentTx.RSMCMultiAddressScriptPubKey = bobRsmcMultiAddressScriptPubKey
	bobCommitmentTx.RSMCTxHex = bobSignedRsmcHex
	bobCommitmentTx.RSMCTxid = bobRsmcTxid
	bobCommitmentTx.AmountToRSMC = latestCcommitmentTxInfo.AmountToCounterparty
	err = createCurrCommitmentTxBR(tx, dao.BRType_Rmsc, channelInfo, bobCommitmentTx, inputs, myChannelAddress, myChannelPrivateKey, *user)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}
	//endregion

	//签名对方传过来的toOtherHex
	_, bobSignedToOtherHex, err := rpcClient.BtcSignRawTransaction(bobToOtherHex, myChannelPrivateKey)
	if err != nil {
		return nil, false, errors.New("fail to sign toOther hex ")
	}
	testResult, err = rpcClient.TestMemPoolAccept(bobSignedToOtherHex)
	if err != nil {
		return nil, false, err
	}
	if gjson.Parse(testResult).Array()[0].Get("allowed").Bool() == false {
		return nil, false, errors.New(gjson.Parse(testResult).Array()[0].Get("reject-reason").String())
	}
	bobData["signedToOtherHex"] = bobSignedToOtherHex

	channelInfo.CurrState = dao.ChannelState_CanUse
	_ = tx.Update(channelInfo)

	_ = tx.Commit()

	//同步通道信息到tracker
	sendChannelStateToTracker(*channelInfo, *latestCcommitmentTxInfo)

	aliceData["channelId"] = channelId
	bobData["channelId"] = channelId

	retData = make(map[string]interface{})
	retData["aliceData"] = aliceData
	retData["bobData"] = bobData
	return retData, true, nil
}

//52 bob保存更新自己的交易
func (service *htlcCloseTxManager) AfterAliceSignCloseHTLCAtBobSide(data string, user *bean.User) (retData map[string]interface{}, err error) {
	jsonObj := gjson.Parse(data)

	var channelId = jsonObj.Get("channelId").String()
	var signedRsmcHex = jsonObj.Get("signedRsmcHex").String()
	var signedToOtherHex = jsonObj.Get("signedToOtherHex").String()
	var rdHex = jsonObj.Get("rdHex").String()

	tx, err := user.Db.Begin(true)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer tx.Rollback()

	channelInfo := &dao.ChannelInfo{}
	err = tx.Select(
		q.Eq("ChannelId", jsonObj.Get("channelId").String()),
		q.Or(
			q.Eq("PeerIdA", user.PeerId),
			q.Eq("PeerIdB", user.PeerId))).
		First(channelInfo)
	if err != nil {
		return nil, errors.New("not found channelInfo at targetSide")
	}

	latestCcommitmentTxInfo, err := getLatestCommitmentTxUseDbTx(tx, channelId, user.PeerId)
	if err != nil {
		err = errors.New("fail to find sender's commitmentTxInfo")
		log.Println(err)
		return nil, err
	}

	if latestCcommitmentTxInfo.CurrState != dao.TxInfoState_Create {
		err = errors.New("wrong commitmentTxInfo state " + strconv.Itoa(int(latestCcommitmentTxInfo.CurrState)))
		log.Println(err)
		return nil, err
	}

	myChannelAddress := channelInfo.AddressB
	if user.PeerId == channelInfo.PeerIdA {
		myChannelAddress = channelInfo.AddressA
	}

	err = signRdTx(tx, channelInfo, signedRsmcHex, rdHex, latestCcommitmentTxInfo, myChannelAddress, user)
	if err != nil {
		return nil, err
	}

	//更新alice的当前承诺交易
	latestCcommitmentTxInfo.SignAt = time.Now()
	latestCcommitmentTxInfo.CurrState = dao.TxInfoState_CreateAndSign
	latestCcommitmentTxInfo.RSMCTxHex = signedRsmcHex
	latestCcommitmentTxInfo.ToCounterpartyTxHex = signedToOtherHex
	bytes, err := json.Marshal(latestCcommitmentTxInfo)
	msgHash := tool.SignMsgWithSha256(bytes)
	latestCcommitmentTxInfo.CurrHash = msgHash
	_ = tx.Update(latestCcommitmentTxInfo)

	lastCommitmentTxInfo := dao.CommitmentTransaction{}
	err = tx.One("Id", latestCcommitmentTxInfo.LastCommitmentTxId, &lastCommitmentTxInfo)
	if err == nil {
		lastCommitmentTxInfo.CurrState = dao.TxInfoState_Abord
		_ = tx.Update(lastCommitmentTxInfo)
	}

	channelInfo.CurrState = dao.ChannelState_CanUse
	_ = tx.Update(channelInfo)

	_ = tx.Commit()

	retData = make(map[string]interface{})
	retData["latestCcommitmentTxInfo"] = latestCcommitmentTxInfo
	return retData, nil
}

func addHT1aTxToWaitDB(htnx *dao.HTLCTimeoutTxForAAndExecutionForB, htrd *dao.RevocableDeliveryTransaction) error {
	node := &dao.RDTxWaitingSend{}
	count, err := db.Select(
		q.Eq("TransactionHex", htnx.RSMCTxHex)).
		Count(node)
	if err == nil {
		return err
	}
	if count > 0 {
		return errors.New("already save")
	}
	node.TransactionHex = htnx.RSMCTxHex
	node.Type = 1
	node.IsEnable = true
	node.CreateAt = time.Now()
	node.HtnxIdAndHtnxRdId = make([]int, 2)
	node.HtnxIdAndHtnxRdId[0] = htnx.Id
	node.HtnxIdAndHtnxRdId[1] = htrd.Id
	err = db.Save(node)
	if err != nil {
		return err
	}
	return nil
}

func addHTRD1aTxToWaitDB(htnxIdAndHtnxRdId []int) error {
	htnxId := htnxIdAndHtnxRdId[0]
	htrdId := htnxIdAndHtnxRdId[1]
	htnx := dao.HTLCTimeoutTxForAAndExecutionForB{}
	err := db.One("Id", htnxId, &htnx)
	if err != nil {
		return err
	}

	htrd := dao.RevocableDeliveryTransaction{}
	err = db.One("Id", htrdId, &htrd)
	if err != nil {
		return err
	}

	node := &dao.RDTxWaitingSend{}
	count, err := db.Select(
		q.Eq("TransactionHex", htrd.TxHex)).
		Count(node)
	if err == nil {
		return err
	}
	if count > 0 {
		return errors.New("already save")
	}

	node.TransactionHex = htrd.TxHex
	node.Type = 0
	node.IsEnable = true
	node.CreateAt = time.Now()
	err = db.Save(node)
	if err != nil {
		return err
	}

	htnx.CurrState = dao.TxInfoState_SendHex
	htnx.SendAt = time.Now()
	_ = db.Update(htnx)

	return nil
}

//htlc timeout Delivery 1b
func addHTDnxTxToWaitDB(txInfo *dao.HTLCTimeoutDeliveryTxB) (err error) {
	node := &dao.RDTxWaitingSend{}
	count, err := db.Select(
		q.Eq("TransactionHex", txInfo.TxHex)).
		Count(node)
	if err == nil {
		return err
	}
	if count > 0 {
		return errors.New("already save")
	}
	node.TransactionHex = txInfo.TxHex
	node.Type = 2
	node.IsEnable = true
	node.CreateAt = time.Now()
	err = db.Save(node)
	if err != nil {
		return err
	}
	return nil
}
