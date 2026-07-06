package server

import (
	"github.com/feenlace/mcp-1c/dump"
	"github.com/feenlace/mcp-1c/onec"
	"github.com/feenlace/mcp-1c/prompts"
	"github.com/feenlace/mcp-1c/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New creates an MCP server with basic configuration and registers tools.
// If dumpIndex is provided, the search_code tool will be registered.
//
// writeClient controls the accounting write tools (create_document,
// post_document, unpost_document, create_catalog_item, update_catalog_item,
// update_document, set_deletion_mark): nil disables them entirely — an LLM
// connected to a server started with writeClient=nil never sees these tools
// in tools/list and therefore cannot call them, which is the primary safety
// boundary for write access (see docs/WRITE-TOOLS.md). When non-nil,
// writeClient is used only for the write tools; it is expected to be
// authenticated as a separate, minimally-privileged 1C user (e.g. mcp_writer)
// distinct from onecClient's read-only credentials, so a misconfigured
// onecClient can never accidentally gain write access.
//
// writeBlacklist is a second, independent safety layer: document/catalog
// type-name glob patterns (see tools.IsWriteBlacklisted) that every write
// tool refuses to touch even when writeClient has full 1C rights. This
// protects e.g. bank-integrated payment documents from AI-issued writes
// regardless of the underlying 1C user's permissions. A nil/empty slice
// disables the check entirely.
func New(version string, onecClient *onec.Client, dumpIndex *dump.Index, writeClient *onec.Client, writeBlacklist []string) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "mcp-1c",
			Version: version,
		},
		nil,
	)
	s.AddTool(tools.MetadataTool(), tools.NewMetadataHandler(onecClient))
	s.AddTool(tools.ObjectStructureTool(), tools.NewObjectStructureHandler(onecClient))
	s.AddTool(tools.QueryTool(), tools.NewQueryHandler(onecClient))
	if dumpIndex != nil {
		s.AddTool(tools.SearchCodeTool(), tools.NewSearchCodeHandler(dumpIndex))
		s.AddTool(tools.ProposeModuleChangeTool(), tools.NewProposeModuleChangeHandler(dumpIndex))
		s.AddTool(tools.ListProposedChangesTool(), tools.NewListProposedChangesHandler(dumpIndex))
	}

	// Pass dump directory to form handler so it can enrich the HTTP response
	// with data from Form.xml files parsed from the dump.
	var dumpDir string
	if dumpIndex != nil {
		dumpDir = dumpIndex.Dir()
	}
	s.AddTool(tools.FormStructureTool(), tools.NewFormStructureHandler(onecClient, dumpDir))

	s.AddTool(tools.ValidateQueryTool(), tools.NewValidateQueryHandler(onecClient))
	s.AddTool(tools.EventLogTool(), tools.NewEventLogHandler(onecClient))
	s.AddTool(tools.ConfigurationInfoTool(), tools.NewConfigurationInfoHandler(onecClient))
	tools.RegisterBSLHelp(s)

	// get_document is read-only (no state change), so it is available
	// regardless of writeClient — useful to inspect any document's state.
	// It uses onecClient: MCP_ОсновнаяРоль grants the ДокументПоСсылке URL
	// template too, so plain read-only credentials are enough.
	s.AddTool(tools.GetDocumentTool(), tools.NewGetDocumentHandler(onecClient))

	// get_catalog_item and find_ref are also read-only and always available,
	// same reasoning as get_document above.
	s.AddTool(tools.GetCatalogItemTool(), tools.NewGetCatalogItemHandler(onecClient))
	s.AddTool(tools.FindRefTool(), tools.NewFindRefHandler(onecClient))

	if writeClient != nil {
		s.AddTool(tools.CreateDocumentTool(), tools.NewCreateDocumentHandler(writeClient, writeBlacklist))
		s.AddTool(tools.PostDocumentTool(), tools.NewPostDocumentHandler(writeClient, writeBlacklist))
		s.AddTool(tools.UnpostDocumentTool(), tools.NewUnpostDocumentHandler(writeClient, writeBlacklist))
		s.AddTool(tools.UpdateDocumentTool(), tools.NewUpdateDocumentHandler(writeClient, writeBlacklist))
		s.AddTool(tools.CreateCatalogItemTool(), tools.NewCreateCatalogItemHandler(writeClient, writeBlacklist))
		s.AddTool(tools.UpdateCatalogItemTool(), tools.NewUpdateCatalogItemHandler(writeClient, writeBlacklist))
		s.AddTool(tools.SetDeletionMarkTool(), tools.NewSetDeletionMarkHandler(writeClient, writeBlacklist))
	}

	prompts.RegisterAll(s)
	return s
}
