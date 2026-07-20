package app

import (
	"context"
	"net/http"
	"strconv"

	"chihqiang/ccsim-svr/config"
	"chihqiang/ccsim-svr/handler"
	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/repo"
	"chihqiang/ccsim-svr/service"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
	gws "github.com/chihqiang/infra-go/websocket"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// App 应用主体，承载依赖注入与生命周期管理
type App struct {
	Cfg    *config.Config
	NodeID string
	DB     *gorm.DB

	// 存储层
	VisitorRepo      *repo.VisitorRepo
	AgentRepo        *repo.AgentRepo
	SessionRepo      *repo.SessionRepo
	MessageRepo      *repo.MessageRepo
	SatisfactionRepo *repo.SatisfactionRatingRepo

	// 服务层
	VisitorService      *service.VisitorService
	AgentService        *service.AgentService
	SessionService      *service.SessionService
	MessageService      *service.MessageService
	SatisfactionService *service.SatisfactionService

	// WebSocket
	Hub    *ws.Hub
	Router *handler.Router
}

// NewApp 创建应用并完成依赖注入
func NewApp(ctx context.Context, cfg *config.Config) *App {
	a := &App{NodeID: uuid.New().String()[:8], Cfg: cfg}
	a.initLogger(ctx)
	a.initDB(ctx)
	a.initRepos()
	a.initServices()
	a.initHub()
	a.initRouter()
	return a
}

func (a *App) initLogger(ctx context.Context) {
	logger.ReplaceGlobal(a.Cfg.Log)
}

func (a *App) initDB(ctx context.Context) {
	db, err := orm.New(a.Cfg.Database)
	if err != nil {
		logger.FatalfCtx(ctx, "连接数据库失败: %v", err)
	}
	a.DB = db

	if err := model.Migrate(db); err != nil {
		logger.FatalfCtx(ctx, "数据库迁移失败: %v", err)
	}
	logger.InfoCtx(ctx, "数据库迁移完成")

	if err := model.Seed(db); err != nil {
		logger.FatalfCtx(ctx, "生成测试数据失败: %v", err)
	}
	logger.InfoCtx(ctx, "测试数据初始化完成")
}

func (a *App) initRepos() {
	_ = repo.NewTenantRepo(a.DB)
	a.VisitorRepo = repo.NewVisitorRepo(a.DB)
	a.AgentRepo = repo.NewAgentRepo(a.DB)
	a.SessionRepo = repo.NewSessionRepo(a.DB)
	a.MessageRepo = repo.NewMessageRepo(a.DB)
	a.SatisfactionRepo = repo.NewSatisfactionRatingRepo(a.DB)
}

func (a *App) initServices() {
	a.VisitorService = service.NewVisitorService(a.VisitorRepo)
	a.AgentService = service.NewAgentService(a.AgentRepo)
	a.SessionService = service.NewSessionService(a.SessionRepo, a.VisitorRepo, a.AgentRepo)
	a.MessageService = service.NewMessageService(a.MessageRepo, a.SessionRepo)
	a.SatisfactionService = service.NewSatisfactionService(a.SatisfactionRepo)
}

func (a *App) initHub() {
	// 基础配置从 yaml 加载，运行时字段覆盖
	nodeID16, _ := strconv.ParseUint(a.NodeID[:4], 16, 16)
	wsCfg := a.Cfg.WS
	wsCfg.NodeID = uint16(nodeID16)

	a.Hub = ws.NewHub(wsCfg, gws.WithCheckOrigin(func(r *http.Request) bool { return true }))

	// standalone 模式下才使用 onAgentOffline 回调写 DB
	a.Hub.SetOnAgentOffline(func(ctx context.Context, agentID int64) {
		if err := a.AgentService.SetOnline(ctx, agentID, false, ""); err != nil {
			logger.ErrorfCtx(ctx, "标记客服离线失败, 客服ID: %d, 错误: %v", agentID, err)
		}
	})
}

func (a *App) initRouter() {
	a.Router = handler.NewRouter()

	authHandler := handler.NewAuthHandler(a.Hub, a.VisitorService, a.AgentService, a.MessageService)
	heartbeatHandler := handler.NewHeartbeatHandler(a.Hub)
	chatHandler := handler.NewChatHandler(a.Hub, a.SessionService, a.MessageService)
	sessionHandler := handler.NewSessionHandler(a.Hub, a.SessionService, a.MessageService, a.AgentService, a.VisitorService)
	agentHandler := handler.NewAgentHandler(a.Hub, a.AgentService, a.SessionService, a.MessageService)
	extraHandler := handler.NewExtraHandler(a.Hub, a.SessionService, a.VisitorService, a.SatisfactionService)

	a.Router.Register(protocol.ClientMsgAuth, authHandler)
	a.Router.Register(protocol.ClientMsgHeartbeat, heartbeatHandler)
	a.Router.Register(protocol.ClientMsgChatSend, chatHandler)
	a.Router.Register(protocol.ClientMsgSessionAccept, sessionHandler)
	a.Router.Register(protocol.ClientMsgSessionClose, sessionHandler)
	a.Router.Register(protocol.ClientMsgSessionList, sessionHandler)
	a.Router.Register(protocol.ClientMsgSessionHistory, sessionHandler)
	a.Router.Register(protocol.ClientMsgWaitingList, sessionHandler)
	a.Router.Register(protocol.ClientMsgAgentOnline, agentHandler)
	a.Router.Register(protocol.ClientMsgAgentOffline, agentHandler)
	a.Router.Register(protocol.ClientMsgTyping, agentHandler)
	a.Router.Register(protocol.ClientMsgMessageRead, agentHandler)
	a.Router.Register(protocol.ClientMsgSatisfactionRate, extraHandler)
	a.Router.Register(protocol.ClientMsgVisitorUpdate, extraHandler)
	a.Hub.SetRouter(a.Router)
}
