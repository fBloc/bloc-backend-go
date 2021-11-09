package mongo

import (
	"context"
	"time"

	"github.com/fBloc/bloc-backend-go/aggregate"
	"github.com/fBloc/bloc-backend-go/internal/conns/mongodb"
	"github.com/fBloc/bloc-backend-go/internal/json_date"
	"github.com/fBloc/bloc-backend-go/repository/flow_run_record"
	"github.com/fBloc/bloc-backend-go/value_object"

	"github.com/google/uuid"
)

const (
	DefaultCollectionName = "flow_run_record"
)

func init() {
	var _ flow_run_record.FlowRunRecordRepository = &MongoRepository{}
}

type MongoRepository struct {
	mongoCollection *mongodb.Collection
}

// Create a new mongodb repository
func New(
	ctx context.Context,
	hosts []string, port int, user, password, db, collectionName string,
) (*MongoRepository, error) {
	collection := mongodb.NewCollection(
		hosts, port, user, password, db, collectionName,
	)
	return &MongoRepository{mongoCollection: collection}, nil
}

type mongoFlowRunRecord struct {
	ID                           uuid.UUID                            `bson:"id"`
	ArrangementID                uuid.UUID                            `bson:"arrangement_id,omitempty"`
	ArrangementFlowID            string                               `bson:"arrangement_flow_id,omitempty"`
	ArrangementRunRecordID       string                               `bson:"arrangement_task_id,omitempty"`
	FlowID                       uuid.UUID                            `bson:"flow_id"`
	FlowOriginID                 uuid.UUID                            `bson:"flow_origin_id"`
	FlowFuncIDMapFuncRunRecordID map[string]uuid.UUID                 `bson:"flowFuncID_map_funcRunRecordID"`
	TriggerTime                  time.Time                            `bson:"trigger_time"`
	TriggerKey                   string                               `bson:"trigger_key"`
	TriggerSource                value_object.FlowTriggeredSourceType `bson:"source_type"`
	TriggerType                  value_object.TriggerType             `bson:"trigger_type"`
	TriggerUserID                uuid.UUID                            `bson:"trigger_user_id"`
	StartTime                    time.Time                            `bson:"start_time,omitempty"`
	EndTime                      time.Time                            `bson:"end_time,omitempty"`
	Status                       value_object.RunState                `bson:"status"`
	ErrorMsg                     string                               `bson:"error_msg,omitempty"`
	RetriedAmount                uint16                               `bson:"retried_amount"`
	TimeoutCanceled              bool                                 `bson:"timeout_canceled,omitempty"`
	Canceled                     bool                                 `bson:"canceled"`
	CancelUserID                 uuid.UUID                            `bson:"cancel_user_id"`
}

func NewFromAggregate(
	fRR *aggregate.FlowRunRecord,
) *mongoFlowRunRecord {
	resp := mongoFlowRunRecord{
		ID:                           fRR.ID,
		ArrangementID:                fRR.ArrangementID,
		ArrangementFlowID:            fRR.ArrangementFlowID,
		ArrangementRunRecordID:       fRR.ArrangementRunRecordID,
		FlowID:                       fRR.FlowID,
		FlowOriginID:                 fRR.FlowOriginID,
		FlowFuncIDMapFuncRunRecordID: fRR.FlowFuncIDMapFuncRunRecordID,
		TriggerTime:                  fRR.TriggerTime,
		TriggerKey:                   fRR.TriggerKey,
		TriggerSource:                fRR.TriggerSource,
		TriggerType:                  fRR.TriggerType,
		TriggerUserID:                fRR.TriggerUserID,
		StartTime:                    fRR.StartTime,
		EndTime:                      fRR.EndTime,
		Status:                       fRR.Status,
		ErrorMsg:                     fRR.ErrorMsg,
		RetriedAmount:                fRR.RetriedAmount,
		TimeoutCanceled:              fRR.TimeoutCanceled,
		Canceled:                     fRR.Canceled,
		CancelUserID:                 fRR.CancelUserID,
	}
	return &resp
}

func (m mongoFlowRunRecord) ToAggregate() *aggregate.FlowRunRecord {
	resp := aggregate.FlowRunRecord{
		ID:                           m.ID,
		ArrangementID:                m.ArrangementID,
		ArrangementFlowID:            m.ArrangementFlowID,
		ArrangementRunRecordID:       m.ArrangementRunRecordID,
		FlowID:                       m.FlowID,
		FlowOriginID:                 m.FlowOriginID,
		FlowFuncIDMapFuncRunRecordID: m.FlowFuncIDMapFuncRunRecordID,
		TriggerTime:                  m.TriggerTime,
		TriggerKey:                   m.TriggerKey,
		TriggerSource:                m.TriggerSource,
		TriggerType:                  m.TriggerType,
		TriggerUserID:                m.TriggerUserID,
		StartTime:                    m.StartTime,
		EndTime:                      m.EndTime,
		Status:                       m.Status,
		ErrorMsg:                     m.ErrorMsg,
		RetriedAmount:                m.RetriedAmount,
		TimeoutCanceled:              m.TimeoutCanceled,
		Canceled:                     m.Canceled,
		CancelUserID:                 m.CancelUserID,
	}
	return &resp
}

// create
func (mr *MongoRepository) Create(fRR *aggregate.FlowRunRecord) error {
	m := NewFromAggregate(fRR)
	_, err := mr.mongoCollection.InsertOne(*m)
	return err
}

// Read
func (mr *MongoRepository) get(
	filter *mongodb.MongoFilter,
) (*aggregate.FlowRunRecord, error) {
	var mFRR mongoFlowRunRecord
	err := mr.mongoCollection.Get(filter, nil, &mFRR)
	if err != nil {
		return nil, err
	}
	return mFRR.ToAggregate(), err
}

func (mr *MongoRepository) GetByID(
	id uuid.UUID,
) (*aggregate.FlowRunRecord, error) {
	return mr.get(mongodb.NewFilter().AddEqual("id", id))
}

func (mr *MongoRepository) ReGetToCheckIsCanceled(
	id uuid.UUID,
) bool {
	aggFRR, err := mr.get(mongodb.NewFilter().AddEqual("id", id))
	if err != nil {
		return false // 访问失败的，保守处理为没有取消
	}
	return aggFRR.Canceled
}

func (mr *MongoRepository) GetLatestByFlowOriginID(
	flowOriginID uuid.UUID,
) (*aggregate.FlowRunRecord, error) {
	return mr.get(mongodb.NewFilter().AddEqual("flow_origin_id", flowOriginID))
}

func (mr *MongoRepository) GetLatestByFlowID(
	flowID uuid.UUID,
) (*aggregate.FlowRunRecord, error) {
	return mr.get(mongodb.NewFilter().AddEqual("flow_id", flowID))
}

func (mr *MongoRepository) GetLatestByArrangementFlowID(
	arrangementFlowID string,
) (*aggregate.FlowRunRecord, error) {
	return mr.get(mongodb.NewFilter().AddEqual("arrangement_flow_id", arrangementFlowID))
}

func (mr *MongoRepository) Filter(
	filter value_object.RepositoryFilter,
	filterOption value_object.RepositoryFilterOption,
) ([]*aggregate.FlowRunRecord, error) {
	var mFRRs []mongoFlowRunRecord
	err := mr.mongoCollection.CommonFilter(filter, filterOption, &mFRRs)
	if err != nil {
		return nil, err
	}

	resp := make([]*aggregate.FlowRunRecord, 0, len(mFRRs))
	for _, i := range mFRRs {
		resp = append(resp, i.ToAggregate())
	}

	return resp, nil
}

// 返回某个flow作为运行源的其全部`运行中`记录
func (mr *MongoRepository) AllRunRecordOfFlowTriggeredByFlowID(
	flowID uuid.UUID,
) ([]*aggregate.FlowRunRecord, error) {
	var mFRRs []mongoFlowRunRecord
	filter := value_object.NewRepositoryFilter()
	filter.AddEqual("flow_id", flowID)

	err := mr.mongoCollection.CommonFilter(
		*filter, *value_object.NewRepositoryFilterOption(), &mFRRs,
	)
	if err != nil {
		return nil, err
	}

	resp := make([]*aggregate.FlowRunRecord, 0, len(mFRRs))
	for _, i := range mFRRs {
		aggRR := i.ToAggregate()
		if aggRR.IsFromArrangement() {
			continue
		}
		resp = append(resp, aggRR)
	}

	return resp, nil
}

// update
func (mr *MongoRepository) PatchDataForRetry(
	id uuid.UUID, retriedAmount uint16,
) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("retried_amount", retriedAmount+1))
}

func (mr *MongoRepository) PatchFlowFuncIDMapFuncRunRecordID(
	id uuid.UUID,
	FlowFuncIDMapFuncRunRecordID map[string]uuid.UUID,
) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("flowFuncID_map_funcRunRecordID", FlowFuncIDMapFuncRunRecordID).
			AddSet("status", value_object.Running))
}

func (mr *MongoRepository) AddFlowFuncIDMapFuncRunRecordID(
	id uuid.UUID,
	flowFuncID string,
	funcRunRecordID uuid.UUID,
) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().AddSet(
			"flowFuncID_map_funcRunRecordID."+flowFuncID,
			funcRunRecordID),
	)
}

func (mr *MongoRepository) Start(id uuid.UUID) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("status", value_object.InQueue).
			AddSet("start_time", time.Now()))
}

func (mr *MongoRepository) Suc(id uuid.UUID) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("status", value_object.Suc).
			AddSet("end_time", time.Now()))
}

func (mr *MongoRepository) Fail(id uuid.UUID, errorMsg string) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("status", value_object.Fail).
			AddSet("error_msg", errorMsg).
			AddSet("end_time", time.Now()))
}

func (mr *MongoRepository) TimeoutCancel(id uuid.UUID) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("status", value_object.TimeoutCanceled).
			AddSet("canceled", true).
			AddSet("end_time", json_date.Now()),
	)
}

func (mr *MongoRepository) UserCancel(id, userID uuid.UUID) error {
	return mr.mongoCollection.PatchByID(
		id,
		mongodb.NewUpdater().
			AddSet("status", value_object.UserCanceled).
			AddSet("canceled", true).
			AddSet("cancel_user_id", userID).
			AddSet("end_time", json_date.Now()),
	)
}

// Delete