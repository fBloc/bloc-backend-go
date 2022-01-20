package bloc

import (
	"log"
	"net/http"

	"github.com/fBloc/bloc-server/event"
	"github.com/fBloc/bloc-server/interfaces/web/bloc_root"
	"github.com/fBloc/bloc-server/interfaces/web/client"
	"github.com/fBloc/bloc-server/interfaces/web/flow"
	"github.com/fBloc/bloc-server/interfaces/web/flow_run_record"
	"github.com/fBloc/bloc-server/interfaces/web/function"
	"github.com/fBloc/bloc-server/interfaces/web/function_run_record"
	"github.com/fBloc/bloc-server/interfaces/web/log_data"
	"github.com/fBloc/bloc-server/interfaces/web/middleware"
	"github.com/fBloc/bloc-server/interfaces/web/object_storage"
	"github.com/fBloc/bloc-server/interfaces/web/user"
	flow_service "github.com/fBloc/bloc-server/services/flow"
	flowRunRecord_service "github.com/fBloc/bloc-server/services/flow_run_record"
	function_service "github.com/fBloc/bloc-server/services/function"
	functionRunRecord_service "github.com/fBloc/bloc-server/services/function_run_record"
	user_service "github.com/fBloc/bloc-server/services/user"
	user_cache "github.com/fBloc/bloc-server/services/userid_cache"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

func (blocApp *BlocApp) RunHttpServer() {
	router := httprouter.New()

	httpLogger := blocApp.GetOrCreateHttpLogger()
	// TODO 放这里合适吗？
	event.InjectMq(blocApp.GetOrCreateEventMQ())

	uCacheService, err := user_cache.NewUserCacheService(
		user_cache.WithLogger(httpLogger),
		user_cache.WithUser(blocApp.GetOrCreateUserRepository()),
	)
	if err != nil {
		panic(err)
	}

	// middleware 依赖资源注入
	middleware.InjectUserIDCacheService(uCacheService)

	// root, 4 live detection ...
	{
		router.GET("/api/v1/bloc", bloc_root.HelloBloc)
	}

	// user
	{
		// initial relied services
		userService, err := user_service.NewUserService(
			user_service.WithLogger(httpLogger),
			user_service.WithUserRepository(blocApp.GetOrCreateUserRepository()))
		if err != nil {
			panic(err)
		}
		user.InjectUserService(userService)

		// 确保默认用户存在（否则没法登录前端、查看功能）
		initialUserName, initialUserPasswd := blocApp.InitialUserInfo()
		user.InitialUserExistOrCreate(initialUserName, initialUserPasswd)

		// router
		router.POST("/api/v1/login", user.LoginHandler)

		basicPath := "/api/v1/user"
		router.GET(basicPath, middleware.LoginAuth(user.FilterByName))
		router.POST(basicPath, middleware.SuperuserAuth(user.AddUser))
		router.DELETE(basicPath+"/delete_by_id/:id", middleware.SuperuserAuth(user.DeleteUser))
	}

	// function
	{
		// initial relied services
		funcService, err := function_service.NewFunctionService(
			function_service.WithLogger(httpLogger),
			function_service.WithFunctionRepository(
				blocApp.GetOrCreateFunctionRepository()),
			function_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		function.InjectFunctionService(funcService)

		// router
		{
			// function本身
			basicPath := "/api/v1/function"
			router.GET(basicPath, middleware.LoginAuth(function.All))
		}
		{
			// function权限
			basicPath := "/api/v1/function_permission"
			router.GET(basicPath, middleware.LoginAuth(function.GetPermissionByFunctionID))
			router.POST(basicPath+"/add_permission", middleware.LoginAuth(function.AddUserPermission))
			router.DELETE(basicPath+"/remove_permission", middleware.LoginAuth(function.DeleteUserPermission))
		}
	}

	// function_run_record
	{
		fRRS, err := functionRunRecord_service.NewService(
			functionRunRecord_service.WithLogger(httpLogger),
			functionRunRecord_service.WithFunctionRunRecordRepository(
				blocApp.GetOrCreateFunctionRunRecordRepository()),
			functionRunRecord_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		function_run_record.InjectFunctionRunRecordService(fRRS)

		logBackEnd, err := blocApp.GetOrCreateLogBackEnd()
		if err != nil {
			panic(err)
		}
		function_run_record.InjectLogCollectBackend(logBackEnd)

		// router
		basicPath := "/api/v1/function_run_record"
		router.GET(basicPath, middleware.LoginAuth(function_run_record.Filter))
		router.GET(basicPath+"/get_by_id/:id", middleware.LoginAuth(function_run_record.Get))
		router.GET(basicPath+"/get_log_by_logkey/:log_key", middleware.LoginAuth(function_run_record.GetLogByKey))
	}

	// flow相关
	{
		// initial relied services
		flowService, err := flow_service.NewFlowService(
			flow_service.WithLogger(httpLogger),
			flow_service.WithFlowRepository(
				blocApp.GetOrCreateFlowRepository(),
			),
			flow_service.WithFunctionRepository(
				blocApp.GetOrCreateFunctionRepository(),
			),
			flow_service.WithFunctionRunRecordRepository(
				blocApp.GetOrCreateFunctionRunRecordRepository(),
			),
			flow_service.WithFlowRunRecordRepository(
				blocApp.GetOrCreateFlowRunRecordRepository(),
			),
			flow_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		flow.InjectFlowService(flowService)

		// config
		{
			// 约定的一些字段
			basicPath := "/api/v1/configs"
			router.GET(basicPath, middleware.LoginAuth(flow.FilterDraftByName))
		}

		// router
		{
			// draft flow
			basicPath := "/api/v1/draft_flow"
			router.GET(basicPath, middleware.LoginAuth(flow.FilterDraftByName))
			router.GET(basicPath+"/get_by_origin_id/:origin_id", middleware.LoginAuth(flow.GetDraftByOriginID))
			router.GET(basicPath+"/commit_by_id/:id", middleware.LoginAuth(flow.PubDraft))
			router.POST(basicPath, middleware.LoginAuth(flow.CreateDraft))
			router.PATCH(basicPath, middleware.LoginAuth(flow.UpdateDraft))
			router.DELETE(
				basicPath+"/delete_by_origin_id/:origin_id",
				middleware.LoginAuth(flow.DeleteDraftByOriginID))
		}

		{
			// online flow
			basicPath := "/api/v1/flow"
			router.GET(basicPath, middleware.LoginAuth(flow.FilterFlow))
			router.GET(basicPath+"/get_by_id/:id", middleware.LoginAuth(flow.GetFlowByID))
			router.GET(basicPath+"/get_by_flow_run_record_id/:flow_run_record_id", middleware.LoginAuth(flow.GetFlowByCertainFlowRunRecord))
			router.GET(basicPath+"/get_latestonline_by_origin_id/:origin_id", middleware.LoginAuth(flow.GetFlowByOriginID))
			router.PATCH(basicPath+"/set_execute_control_attributes", middleware.LoginAuth(flow.SetExecuteControlAttributes))
			router.DELETE(basicPath+"delete_by_origin_id/:origin_id", middleware.LoginAuth(flow.DeleteFlowByOriginID))
		}

		{
			// 运行相关
			basicPath := "/api/v1/flow"
			router.GET(basicPath+"/run/by_origin_id/:origin_id", middleware.LoginAuth(flow.Run))
			router.GET(basicPath+"/cancel_run/by_origin_id/:origin_id", middleware.LoginAuth(flow.CancelRun))
		}

		{
			// 权限
			basicPath := "/api/v1/flow_permission"
			router.GET(basicPath, middleware.LoginAuth(flow.GetPermission))
			router.POST(basicPath+"/add_permission", middleware.LoginAuth(flow.AddUserPermission))
			router.DELETE(basicPath+"/remove_permission", middleware.LoginAuth(flow.DeleteUserPermission))
		}
	}

	// flow_run_record
	{
		// initial relied services
		flowRunRecordService, err := flowRunRecord_service.NewService(
			flowRunRecord_service.WithLogger(httpLogger),
			flowRunRecord_service.WithFlowRunRecordRepository(
				blocApp.GetOrCreateFlowRunRecordRepository()),
			flowRunRecord_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		flow_run_record.InjectFlowRunRecordService(flowRunRecordService)

		// router
		basicPath := "/api/v1/flow_run_record"
		router.GET(basicPath, middleware.LoginAuth(flow_run_record.Filter))
	}

	// object storage
	{
		object_storage.InjectObjectStorageImplement(
			blocApp.GetOrCreateConsumerObjectStorage(),
		)
		{
			basicPath := "/api/v1/object_storage"
			router.GET(basicPath+"/get_string_value_by_key/:key", object_storage.GetValueByKeyReturnString)
		}
	}

	// log
	{
		logBackEnd, err := blocApp.GetOrCreateLogBackEnd()
		if err != nil {
			panic(err)
		}
		log_data.InjectLogCollectBackend(logBackEnd)
		{
			basicPath := "/api/v1/log"
			router.POST(basicPath+"/pull_log_between_time", log_data.PullLog)
			router.GET(basicPath+"/pull_functionRunRecordLog_by_id/:function_run_record_id", log_data.PullFunctionRunRecordLog)
		}
	}

	// function provider client
	{

		funcService, err := function_service.NewFunctionService(
			function_service.WithLogger(httpLogger),
			function_service.WithFunctionRepository(
				blocApp.GetOrCreateFunctionRepository()),
			function_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		client.InjectFunctionService(funcService)
		logBackEnd, err := blocApp.GetOrCreateLogBackEnd()
		if err != nil {
			panic(err)
		}
		client.InjectLogBackend(logBackEnd)
		fRRS, err := functionRunRecord_service.NewService(
			functionRunRecord_service.WithLogger(httpLogger),
			functionRunRecord_service.WithFunctionRunRecordRepository(
				blocApp.GetOrCreateFunctionRunRecordRepository()),
			functionRunRecord_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		client.InjectFunctionRunRecordService(fRRS)

		flowService, err := flow_service.NewFlowService(
			flow_service.WithLogger(httpLogger),
			flow_service.WithFlowRepository(
				blocApp.GetOrCreateFlowRepository(),
			),
			flow_service.WithFunctionRepository(
				blocApp.GetOrCreateFunctionRepository(),
			),
			flow_service.WithFunctionRunRecordRepository(
				blocApp.GetOrCreateFunctionRunRecordRepository(),
			),
			flow_service.WithFlowRunRecordRepository(
				blocApp.GetOrCreateFlowRunRecordRepository(),
			),
			flow_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		client.InjectFlowService(flowService)

		client.InjectConsumerLogger(blocApp.GetOrCreateScheduleLogger())

		flowRunRecordService, err := flowRunRecord_service.NewService(
			flowRunRecord_service.WithLogger(httpLogger),
			flowRunRecord_service.WithFlowRunRecordRepository(
				blocApp.GetOrCreateFlowRunRecordRepository()),
			flowRunRecord_service.WithUserCacheService(uCacheService),
		)
		if err != nil {
			panic(err)
		}
		client.InjectFlowRunRecordService(flowRunRecordService)

		client.InjectObjectStorageImplement(
			blocApp.GetOrCreateConsumerObjectStorage(),
		)

		basicPath := "/api/v1/client"
		{
			router.POST(basicPath+"/register_functions", client.RegisterFunctions)
			router.POST(basicPath+"/report_log", client.ReportLog)
			router.POST(basicPath+"/report_progress", client.ReportProgress)
			router.POST(basicPath+"/persist_certain_function_run_opt_field", client.PersistFuncRunOptField)
			router.POST(basicPath+"/function_run_finished", client.FunctionRunFinished)
			router.GET(basicPath+"/get_function_run_record_by_id/:id", function_run_record.Get)
			router.GET(basicPath+"/check_flowRun_is_canceled_by_flowRunID/:id", client.FlowRunRecordIsCanceled)
			router.GET(basicPath+"/get_byte_value_by_key/:key", object_storage.GetValueByKeyReturnByte)
		}
	}

	// start http server
	log.Printf("start http server at http://%s", blocApp.HttpAddress())
	handler := cors.AllowAll().Handler(router)
	log.Fatal(http.Serve(blocApp.HttpListener(), handler))
}
