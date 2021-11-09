package aggregate

import (
	"time"

	"github.com/fBloc/bloc-backend-go/event"

	"github.com/google/uuid"
)

type IptBriefAndKey struct {
	Brief   interface{}
	FullKey string
}

type FunctionRunRecord struct {
	ID                uuid.UUID
	FlowID            uuid.UUID
	FlowOriginID      uuid.UUID
	ArrangementFlowID string
	FunctionID        uuid.UUID
	FlowFunctionID    string // TODO 改为flow_function_id
	FlowRunRecordID   uuid.UUID
	Start             time.Time
	End               time.Time
	Suc               bool
	Pass              bool
	Canceled          bool
	Description       string
	ErrorMsg          string
	Ipts              [][]interface{}    // 实际调用user implement function的时候，根据用户前端输入的匹配规则/值，获取到实际值填充进此字段，作为参数传递给函数
	IptBriefAndObskey [][]IptBriefAndKey // Ipts中的值可能非常大，在前端显示的时候不可能全部显示，故进行截断，将真实值保存到对象存储，需要的时候才查看真实值
	Opt               map[string]interface{}
	OptBrief          map[string]string
	Progress          float32
	ProgressMsg       []string
	ProcessStages     []string
	ProcessStageIndex int
}

func NewFunctionRunRecordFromFlowDriven(
	functionIns Function, flowRunRecordIns FlowRunRecord,
	flowFunctionID string,
) *FunctionRunRecord {
	fRR := &FunctionRunRecord{
		ID:              uuid.New(),
		FlowID:          flowRunRecordIns.FlowID,
		FlowOriginID:    flowRunRecordIns.FlowOriginID,
		FunctionID:      functionIns.ID,
		FlowFunctionID:  flowFunctionID,
		FlowRunRecordID: flowRunRecordIns.ID,
		Start:           time.Now(),
		ProcessStages:   functionIns.ProcessStages,
	}
	event.PubEvent(&event.FunctionToRun{FunctionRunRecordID: fRR.ID})
	return fRR
}

func (bh *FunctionRunRecord) IsZero() bool {
	if bh == nil {
		return true
	}
	return bh.ID == uuid.Nil
}

func (bh *FunctionRunRecord) UsedSeconds() float64 {
	end := bh.End
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(bh.Start).Seconds()
}

func (bh *FunctionRunRecord) Failed() bool {
	if bh.IsZero() {
		return false
	}
	if bh.End.IsZero() {
		return false
	}
	return bh.Suc
}

func (bh *FunctionRunRecord) Finished() bool {
	if bh.IsZero() {
		return false
	}
	if bh.End.IsZero() {
		return false
	}
	return true
}

func (bh *FunctionRunRecord) SetSuc() {
	bh.Suc = true
}

func (bh *FunctionRunRecord) SetFail(errorMsg string) {
	bh.Suc = false
	bh.ErrorMsg = errorMsg
}