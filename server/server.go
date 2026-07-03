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
// post_document, unpost_document): nil disables them entirely — an LLM
// connected to a server started with writeClient=nil never sees these tools
// in tools/list and therefore cannot call them, which is the primary safety
// boundary for write access (see docs/WRITE-TOOLS.md). When non-nil,
// writeClient is used only for the three write tools; it is expected to be
// authenticated as a separate, minimally-privileged 1C user (e.g. mcp_writer)
// distinct from onecClient's read-only credentials, so a misconfigured
// onecClient can never accidentally gain write access.
func New(version string, onecClient *onec.Client, dumpIndex *dump.Index, writeClient *onec.Client) *mcp.Server {
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

	if writeClient != nil {
		s.AddTool(tools.CreateDocumentTool(), tools.NewCreateDocumentHandler(writeClient))
		s.AddTool(tools.PostDocumentTool(), tools.NewPostDocumentHandler(writeClient))
		s.AddTool(tools.UnpostDocumentTool(), tools.NewUnpostDocumentHandler(writeClient))
	}

	prompts.RegisterAll(s)
	return s
}
