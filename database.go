package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const defaultTransactionTimeout = time.Second * 3

type Database struct {
	pool                           *pgxpool.Pool
	ActiveLanguage                 *ActiveLanguage
	AuditLog                       *AuditLogTable
	ArchiveMessages                *ArchiveMessages
	AutoCloseExclude               *AutoCloseExclude
	Blacklist                      *Blacklist
	BotStaff                       *BotStaff
	CategoryUpdateQueue            *CategoryUpdateQueue
	ChannelCategory                *ChannelCategory
	ClaimSettings                  *ClaimSettingsTable
	CloseReason                    *CloseMetadataTable
	CloseRequest                   *CloseRequestTable
	CustomIntegrations             *CustomIntegrationTable
	CustomIntegrationGuildCounts   *CustomIntegrationGuildCountsView
	CustomIntegrationGuilds        *CustomIntegrationGuildsTable
	CustomIntegrationHeaders       *CustomIntegrationHeadersTable
	CustomIntegrationPlaceholders  *CustomIntegrationPlaceholdersTable
	CustomIntegrationSecretValues  *CustomIntegrationSecretValuesTable
	CustomIntegrationSecrets       *CustomIntegrationSecretsTable
	CustomColours                  *CustomColours
	DashboardOnboarding            *DashboardOnboardingTable
	DashboardUsers                 *DashboardUsersTable
	ArchiveDmMessages              *ArchiveDmMessages
	DiscordEntitlements            *DiscordEntitlements
	DiscordStoreSkus               *DiscordStoreSkus
	EmbedFields                    *EmbedFieldsTable
	Embeds                         *EmbedsTable
	Entitlements                   *Entitlements
	ExitSurveyResponses            *ExitSurveyResponses
	Experiment                     *ExperimentTable
	FirstResponseTime              *FirstResponseTime
	FormInput                      *FormInputTable
	FormInputOption                *FormInputOptionTable
	Forms                          *FormsTable
	FormInputApiConfig             *FormInputApiConfigTable
	FormInputApiHeaders            *FormInputApiHeaderTable
	GalleryListings                *GalleryListingsTable
	GalleryListingTags             *GalleryListingTagsTable
	GdprLogs                       *GDPRLogsTable
	GlobalBlacklist                *GlobalBlacklist
	GuildLeaveTime                 *GuildLeaveTime
	GuildMetadata                  *GuildMetadataTable
	KBArticles                     *KBArticlesTable
	KBCategories                   *KBCategoriesTable
	KBSettings                     *KBSettingsTable
	LegacyPremiumEntitlementGuilds *LegacyPremiumEntitlementGuilds
	LegacyPremiumEntitlements      *LegacyPremiumEntitlements
	MultiPanels                    *MultiPanelTable
	MultiPanelTargets              *MultiPanelTargets
	MultiServerSkus                *MultiServerSkus
	OnCall                         *OnCall
	Panel                          *PanelTable
	PanelAccessControlRules        *PanelAccessControlRules
	PanelKBCategories              *PanelKBCategoriesTable
	PanelRoleMentions              *PanelRoleMentions
	PanelSupportHours              *PanelSupportHoursTable
	PanelSupportHoursSettings      *PanelSupportHoursSettingsTable
	PanelTeams                     *PanelTeamsTable
	PanelTicketPermissions         *PanelTicketPermissionsTable
	PanelAutoClose                 *PanelAutoCloseTable
	PanelUserMention               *PanelUserMention
	PanelHereMention               *PanelHereMention
	Participants                   *ParticipantTable
	PatreonEntitlements            *PatreonEntitlements
	PolarEntitlements              *PolarEntitlements
	PolarProducts                  *PolarProducts
	Permissions                    *Permissions
	PremiumGuilds                  *PremiumGuilds
	PremiumKeys                    *PremiumKeys
	RoleBlacklist                  *RoleBlacklist
	RolePermissions                *RolePermissions
	ServerBlacklist                *ServerBlacklist
	ServiceRatings                 *ServiceRatings
	Settings                       *SettingsTable
	Skus                           *Skus
	StaffOverride                  *StaffOverride
	SubscriptionSkus               *SubscriptionSkus
	SupportTeam                    *SupportTeamTable
	SupportTeamMembers             *SupportTeamMembersTable
	SupportTeamPermissions         *SupportTeamPermissionsTable
	SupportTeamRoles               *SupportTeamRolesTable
	Tag                            *TagsTable
	TicketClaims                   *TicketClaims
	TicketLastMessage              *TicketLastMessageTable
	TicketMembers                  *TicketMembers
	Tickets                        *TicketTable
	UsedKeys                       *UsedKeys
	UserGuilds                     *UserGuildsTable
	VoteCredits                    *VoteCredits
	Votes                          *Votes
	Webhooks                       *WebhookTable
	TicketLabels               *TicketLabelsTable
	TicketLabelAssignments     *TicketLabelAssignmentsTable
	Whitelabel                     *WhitelabelBotTable
	WhitelabelErrors               *WhitelabelErrors
	WhitelabelGuilds               *WhitelabelGuilds
	WhitelabelStatuses             *WhitelabelStatuses
	WhitelabelUsers                *WhitelabelUsers
}

func NewDatabase(pool *pgxpool.Pool) *Database {
	db := &Database{
		pool:                           pool,
		ActiveLanguage:                 newActiveLanguage(pool),
		AuditLog:                       newAuditLogTable(pool),
		ArchiveMessages:                newArchiveMessages(pool),
		AutoCloseExclude:               newAutoCloseExclude(pool),
		Blacklist:                      newBlacklist(pool),
		BotStaff:                       newBotStaff(pool),
		CategoryUpdateQueue:            newCategoryUpdateQueueTable(pool),
		ChannelCategory:                newChannelCategory(pool),
		ClaimSettings:                  newClaimSettingsTable(pool),
		CloseReason:                    newCloseReasonTable(pool),
		CloseRequest:                   newCloseRequestTable(pool),
		CustomIntegrations:             newCustomIntegrationTable(pool),
		CustomIntegrationGuildCounts:   newCustomIntegrationGuildCountsView(pool),
		CustomIntegrationGuilds:        newCustomIntegrationGuildsTable(pool),
		CustomIntegrationHeaders:       newCustomIntegrationHeadersTable(pool),
		CustomIntegrationPlaceholders:  newCustomIntegrationPlaceholdersTable(pool),
		CustomIntegrationSecretValues:  newCustomIntegrationSecretValuesTable(pool),
		CustomIntegrationSecrets:       newCustomIntegrationSecretsTable(pool),
		CustomColours:                  newCustomColours(pool),
		DashboardOnboarding:            newDashboardOnboardingTable(pool),
		DashboardUsers:                 newDashboardUsersTable(pool),
		ArchiveDmMessages:              newArchiveDmMessages(pool),
		DiscordEntitlements:            newDiscordEntitlementsTable(pool),
		DiscordStoreSkus:               newDiscordStoreSkusTable(pool),
		EmbedFields:                    newEmbedFieldsTable(pool),
		Embeds:                         newEmbedsTable(pool),
		Entitlements:                   newEntitlementsTable(pool),
		ExitSurveyResponses:            newExitSurveyResponses(pool),
		Experiment:                     newExperimentTable(pool),
		FirstResponseTime:              newFirstResponseTime(pool),
		FormInput:                      newFormInputTable(pool),
		Forms:                          newFormsTable(pool),
		FormInputApiConfig:             newFormInputApiConfigTable(pool),
		FormInputApiHeaders:            newFormInputApiHeaderTable(pool),
		FormInputOption:                newFormInputOptionTable(pool),
		GalleryListings:                newGalleryListingsTable(pool),
		GalleryListingTags:             newGalleryListingTagsTable(pool),
		GdprLogs:                       newGDPRLogs(pool),
		GlobalBlacklist:                newGlobalBlacklist(pool),
		GuildLeaveTime:                 newGuildLeaveTime(pool),
		GuildMetadata:                  newGuildMetadataTable(pool),
		KBArticles:                     newKBArticles(pool),
		KBCategories:                   newKBCategories(pool),
		KBSettings:                     newKBSettings(pool),
		LegacyPremiumEntitlementGuilds: newLegacyPremiumEntitlementGuildsTable(pool),
		LegacyPremiumEntitlements:      newLegacyPremiumEntitlement(pool),
		MultiPanels:                    newMultiMultiPanelTable(pool),
		MultiPanelTargets:              newMultiPanelTargets(pool),
		MultiServerSkus:                newMultiServerSkusTable(pool),
		OnCall:                         newOnCall(pool),
		Panel:                          newPanelTable(pool),
		PanelAccessControlRules:        newPanelAccessControlRules(pool),
		PanelKBCategories:              newPanelKBCategories(pool),
		PanelRoleMentions:              newPanelRoleMentions(pool),
		PanelSupportHours:              newPanelSupportHoursTable(pool),
		PanelSupportHoursSettings:      newPanelSupportHoursSettingsTable(pool),
		PanelTeams:                     newPanelTeamsTable(pool),
		PanelTicketPermissions:         newPanelTicketPermissionsTable(pool),
		PanelAutoClose:                 newPanelAutoCloseTable(pool),
		PanelUserMention:               newPanelUserMention(pool),
		PanelHereMention:               newPanelHereMention(pool),
		Participants:                   newParticipantTable(pool),
		PatreonEntitlements:            newPatreonEntitlements(pool),
		PolarEntitlements:              newPolarEntitlements(pool),
		PolarProducts:                  newPolarProducts(pool),
		Permissions:                    newPermissions(pool),
		PremiumGuilds:                  newPremiumGuilds(pool),
		PremiumKeys:                    newPremiumKeys(pool),
		RoleBlacklist:                  newRoleBlacklist(pool),
		RolePermissions:                newRolePermissions(pool),
		ServerBlacklist:                newServerBlacklist(pool),
		ServiceRatings:                 newServiceRatings(pool),
		Settings:                       newSettingsTable(pool),
		Skus:                           newSkusTable(pool),
		StaffOverride:                  newStaffOverride(pool),
		SubscriptionSkus:               newSubscriptionSkusTable(pool),
		SupportTeam:                    newSupportTeamTable(pool),
		SupportTeamMembers:             newSupportTeamMembersTable(pool),
		SupportTeamPermissions:         newSupportTeamPermissionsTable(pool),
		SupportTeamRoles:               newSupportTeamRolesTable(pool),
		Tag:                            newTag(pool),
		TicketClaims:                   newTicketClaims(pool),
		TicketLastMessage:              newTicketLastMessageTable(pool),
		TicketMembers:                  newTicketMembers(pool),
		Tickets:                        newTicketTable(pool),
		UsedKeys:                       newUsedKeys(pool),
		UserGuilds:                     newUserGuildsTable(pool),
		VoteCredits:                    newVoteCreditsTable(pool),
		Votes:                          newVotes(pool),
		Webhooks:                       newWebhookTable(pool),
		TicketLabels:               newTicketLabelsTable(pool),
		TicketLabelAssignments:     newTicketLabelAssignmentsTable(pool),
		Whitelabel:                     newWhitelabelBotTable(pool),
		WhitelabelErrors:               newWhitelabelErrors(pool),
		WhitelabelGuilds:               newWhitelabelGuilds(pool),
		WhitelabelStatuses:             newWhitelabelStatuses(pool),
		WhitelabelUsers:                newWhitelabelUsers(pool),
	}

	return db
}

func (d *Database) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return d.pool.Begin(ctx)
}

func (d *Database) WithTx(ctx context.Context, f func(tx pgx.Tx) error) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTransactionTimeout)
		defer cancel()

		tx.Rollback(ctx)
	}()

	if err := f(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (d *Database) CreateTables(ctx context.Context, pool *pgxpool.Pool) {
	mustCreate(ctx, pool,
		d.ActiveLanguage,
		d.Blacklist,
		d.BotStaff,
		d.ChannelCategory,
		d.ClaimSettings,
		d.CustomIntegrations,
		d.CustomIntegrationGuilds,
		d.CustomIntegrationGuildCounts,
		d.CustomIntegrationHeaders,
		d.CustomIntegrationPlaceholders,
		d.CustomIntegrationSecrets,
		d.CustomIntegrationSecretValues,
		d.CustomColours,
		d.DashboardUsers,
		d.Embeds,
		d.EmbedFields, // depends on embeds
		d.Entitlements,
		d.Experiment,
		d.DiscordEntitlements, // depends on entitlements
		d.Skus,                // must be created before discord_store_skus, subscription_skus, multi_server_skus, polar_products
		d.DiscordStoreSkus,    // depends on skus
		d.SubscriptionSkus,    // depends on skus
		d.Forms,
		d.FormInput,           // depends on forms
		d.FormInputOption,     // depends on form inputs
		d.FormInputApiConfig,  // depends on form inputs
		d.FormInputApiHeaders, // depends on form input api config
		d.GdprLogs,
		d.GlobalBlacklist,
		d.GuildLeaveTime,
		d.DashboardOnboarding,
		d.GuildMetadata,
		d.LegacyPremiumEntitlements,
		d.LegacyPremiumEntitlementGuilds,
		d.MultiPanels,
		d.MultiServerSkus,
		d.OnCall,
		d.Panel,
		d.PanelTicketPermissions, // must be created after panels table
		d.PanelAccessControlRules, // must be created after panels table
		d.MultiPanelTargets,       // must be created after panels table
		d.PanelRoleMentions,
		d.PanelSupportHours,         // must be created after panels table
		d.PanelSupportHoursSettings, // must be created after panels table
		d.PanelAutoClose,          // must be created after panels table
		d.PanelUserMention,
		d.PanelHereMention,
		d.PatreonEntitlements,
		d.PolarEntitlements, // depends on entitlements
		d.PolarProducts,    // depends on skus
		d.Permissions,
		d.PremiumGuilds,
		d.PremiumKeys,
		d.RoleBlacklist,
		d.RolePermissions,
		d.ServerBlacklist,
		d.Settings,
		d.StaffOverride,
		d.SupportTeam,
		d.SupportTeamMembers,
		d.SupportTeamRoles,
		d.SupportTeamPermissions, // must be created after support_team table
		d.PanelTeams,             // Must be created after panels & support teams tables
		d.Tag,
		d.KBCategories,        // Knowledge base categories
		d.KBArticles,          // Knowledge base articles (references categories)
		d.KBSettings,          // Knowledge base customisation settings
		d.PanelKBCategories,   // Panel-to-KB-category associations
		d.Tickets,             // Must be created before members table
		d.TicketLastMessage,   // Must be created after Tickets table
		d.Participants,        // Must be created after Tickets table
		d.AutoCloseExclude,    // Must be created after Tickets table
		d.CloseReason,         // Must be created after Tickets table
		d.CloseRequest,        // Must be created after Tickets table
		d.ServiceRatings,      // Must be created after Tickets table
		d.ExitSurveyResponses, // Must be created after Tickets table
		d.ArchiveMessages,     // Must be created after Tickets table
		d.ArchiveDmMessages,   // Must be created after Tickets table
		d.CategoryUpdateQueue, // Must be created after Tickets table
		d.TicketLabels,            // Must be created after Tickets table
		d.TicketLabelAssignments,  // Must be created after Tickets and TicketLabels tables
		d.GalleryListings,         // Gallery panel listings
		d.GalleryListingTags,      // Must be created after GalleryListings table
		d.FirstResponseTime,
		d.TicketMembers,
		d.TicketClaims,
		d.UsedKeys,
		d.UserGuilds,
		d.VoteCredits,
		d.Votes,
		d.Webhooks,
		d.Whitelabel,
		d.WhitelabelErrors,
		d.WhitelabelGuilds,
		d.WhitelabelStatuses,
		d.WhitelabelUsers,
		d.AuditLog,
	)
}

func (d *Database) Views() []View {
	return []View{
		d.CustomIntegrationGuildCounts,
	}
}

func mustCreate(ctx context.Context, pool *pgxpool.Pool, tables ...Table) {
	for _, table := range tables {
		if _, err := pool.Exec(ctx, table.Schema()); err != nil {
			panic(err)
		}
	}
}
