package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/feenlace/mcp-1c/dump"
	"github.com/feenlace/mcp-1c/onec"
	"github.com/feenlace/mcp-1c/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mock1CHandler simulates the 1C HTTP service endpoints.
func mock1CHandler() http.Handler {
	metadata := map[string][]string{
		"Справочники":            {"Контрагенты", "Номенклатура"},
		"Документы":              {"РеализацияТоваровУслуг"},
		"Перечисления":           {"СтавкиНДС", "ВидыНоменклатуры"},
		"Обработки":              {"ЗагрузкаДанныхИзФайла"},
		"Отчеты":                 {"ОборотноСальдоваяВедомость"},
		"РегистрыСведений":       {"КурсыВалют"},
		"РегистрыНакопления":     {"ТоварыНаСкладах"},
		"РегистрыБухгалтерии":    {},
		"ПланыСчетов":            {"Хозрасчетный"},
		"ПланыВидовХарактеристик": {"ВидыСубконтоХозрасчетные"},
		"ПланыОбмена":            {"ОбменБухгалтерия"},
		"ЖурналыДокументов":      {"ЖурналОпераций"},
		"Константы":              {"ОсновнаяОрганизация"},
		"ОбщиеМодули":            {"ОбщийМодуль1"},
		"Роли":                   {"Администратор", "Бухгалтер"},
		"Подсистемы":             {"Бухгалтерия"},
		"HTTPСервисы":            {"MCPService"},
	}

	objects := map[string]onec.ObjectStructure{
		"Document/РеализацияТоваровУслуг": {
			Name:    "РеализацияТоваровУслуг",
			Synonym: "Реализация (акты, накладные, УПД)",
			Attributes: []onec.Attribute{
				{Name: "Контрагент", Synonym: "Контрагент", Type: "СправочникСсылка.Контрагенты"},
				{Name: "СуммаДокумента", Synonym: "Сумма", Type: "Число"},
			},
			TabularParts: []onec.TabularPart{
				{
					Name: "Товары",
					Attributes: []onec.Attribute{
						{Name: "Номенклатура", Synonym: "Номенклатура", Type: "СправочникСсылка.Номенклатура"},
						{Name: "Количество", Synonym: "Количество", Type: "Число"},
					},
				},
			},
		},
		"AccumulationRegister/ТоварыНаСкладах": {
			Name:    "ТоварыНаСкладах",
			Synonym: "Товары на складах",
			Dimensions: []onec.Attribute{
				{Name: "Номенклатура", Synonym: "Номенклатура", Type: "СправочникСсылка.Номенклатура"},
				{Name: "Склад", Synonym: "Склад", Type: "СправочникСсылка.Склады"},
			},
			Resources: []onec.Attribute{
				{Name: "Количество", Synonym: "Количество", Type: "Число"},
			},
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(metadata)
	})

	mux.HandleFunc("/object/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/object/")
		obj, ok := objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"Object not found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(obj)
	})

	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"columns":   []string{"Наименование"},
			"rows":      [][]any{{"ООО Ромашка"}, {"ИП Иванов"}},
			"total":     2,
			"truncated": false,
		})
	})

	mux.HandleFunc("/form/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"name":  "ФормаДокумента",
			"title": "Реализация товаров и услуг",
			"elements": []map[string]any{
				{"name": "Контрагент", "type": "ПолеВвода", "title": "Контрагент", "dataPath": "Объект.Контрагент"},
				{"name": "СуммаДокумента", "type": "ПолеВвода", "title": "Сумма", "dataPath": "Объект.СуммаДокумента"},
			},
			"commands": []map[string]any{
				{"name": "Провести", "action": "ПровестиИЗакрыть"},
			},
			"handlers": []map[string]any{
				{"event": "ПриОткрытии", "handler": "ПриОткрытии"},
			},
		})
	})

	mux.HandleFunc("/eventlog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{
					"date":     "2026-03-07T10:00:00",
					"level":    "Ошибка",
					"event":    "Данные.Запись",
					"user":     "Администратор",
					"metadata": "Документ.РеализацияТоваровУслуг",
					"comment":  "Ошибка при записи документа",
				},
				{
					"date":  "2026-03-07T09:30:00",
					"level": "Информация",
					"event": "Сеанс.Начало",
					"user":  "Бухгалтер",
				},
			},
			"total": 2,
		})
	})

	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"name":             "БухгалтерияПредприятия",
			"version":          "3.0.150.1",
			"vendor":           "Фирма \"1С\"",
			"platform_version": "8.3.25.1000",
			"mode":             "file",
		})
	})

	// In-memory document store for the write-tool integration tests
	// (create_document / post_document / unpost_document / get_document).
	// A magic Организация value ("FORCE_ERROR") makes /document/post return a
	// structured posting_failed error, so tests can exercise that path without
	// a real 1C posting-rule violation.
	type mockDoc struct {
		typ             string
		number          string
		date            string
		posted          bool
		deletionMark    bool
		attributes      map[string]any
		tabularSections map[string][]map[string]any
	}
	docs := map[string]*mockDoc{}
	docSeq := 0

	// In-memory catalog item store for the catalog-tool integration tests
	// (create_catalog_item / update_catalog_item / get_catalog_item).
	type mockCatalogItem struct {
		typ             string
		code            string
		description     string
		isGroup         bool
		deletionMark    bool
		parent          string
		owner           string
		attributes      map[string]any
		tabularSections map[string][]map[string]any
	}
	catalogItems := map[string]*mockCatalogItem{}
	catalogSeq := 0

	mux.HandleFunc("/document/post", func(w http.ResponseWriter, r *http.Request) {
		writeJSONErr := func(status int, code, msg string) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]any{"error": code, "message": msg})
		}
		var req struct {
			Type string `json:"type"`
			Ref  string `json:"ref"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		doc, ok := docs[req.Ref]
		if !ok {
			writeJSONErr(http.StatusNotFound, "not_found", "Документ не найден: "+req.Ref)
			return
		}
		if org, _ := doc.attributes["Организация"].(string); org == "FORCE_ERROR" {
			writeJSONErr(http.StatusBadRequest, "posting_failed", "Недостаточно остатков на складе")
			return
		}
		doc.posted = true
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{"ref": req.Ref, "posted": true, "date": doc.date})
	})

	mux.HandleFunc("/document/unpost", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type string `json:"type"`
			Ref  string `json:"ref"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		doc, ok := docs[req.Ref]
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Документ не найден: " + req.Ref})
			return
		}
		doc.posted = false
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{"ref": req.Ref, "posted": false, "date": doc.date})
	})

	mux.HandleFunc("/document/", func(w http.ResponseWriter, r *http.Request) {
		// GET /document/{type}/{ref}
		path := strings.TrimPrefix(r.URL.Path, "/document/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ref := parts[1]
		doc, ok := docs[ref]
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Документ не найден: " + ref})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"ref": ref, "type": doc.typ, "number": doc.number, "date": doc.date,
			"posted": doc.posted, "deletion_mark": doc.deletionMark,
			"attributes": doc.attributes, "tabular_sections": doc.tabularSections,
		})
	})

	mux.HandleFunc("/document/update", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type            string                      `json:"type"`
			Ref             string                      `json:"ref"`
			Attributes      map[string]any              `json:"attributes"`
			TabularSections map[string][]map[string]any `json:"tabular_sections"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		doc, ok := docs[req.Ref]
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Документ не найден: " + req.Ref})
			return
		}
		if doc.attributes == nil {
			doc.attributes = map[string]any{}
		}
		for k, v := range req.Attributes {
			doc.attributes[k] = v
		}
		if doc.tabularSections == nil {
			doc.tabularSections = map[string][]map[string]any{}
		}
		for k, v := range req.TabularSections {
			doc.tabularSections[k] = v
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"ref": req.Ref, "type": doc.typ, "number": doc.number, "date": doc.date,
			"posted": doc.posted, "deletion_mark": doc.deletionMark,
			"attributes": doc.attributes, "tabular_sections": doc.tabularSections,
		})
	})

	mux.HandleFunc("/document", func(w http.ResponseWriter, r *http.Request) {
		// POST /document
		var req struct {
			Type            string                      `json:"type"`
			Attributes      map[string]any              `json:"attributes"`
			TabularSections map[string][]map[string]any `json:"tabular_sections"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		docSeq++
		ref := fmt.Sprintf("00000000-0000-0000-0000-%012d", docSeq)
		docs[ref] = &mockDoc{
			typ:             req.Type,
			number:          fmt.Sprintf("ЗН-%06d", docSeq),
			date:            "2026-07-02T10:00:00",
			posted:          false,
			attributes:      req.Attributes,
			tabularSections: req.TabularSections,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"ref": ref, "number": docs[ref].number, "date": docs[ref].date, "posted": false,
		})
	})

	mux.HandleFunc("/catalog", func(w http.ResponseWriter, r *http.Request) {
		// POST /catalog
		var req struct {
			Type            string                      `json:"type"`
			IsGroup         bool                        `json:"is_group"`
			OwnerType       string                      `json:"owner_type"`
			Attributes      map[string]any              `json:"attributes"`
			TabularSections map[string][]map[string]any `json:"tabular_sections"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		catalogSeq++
		ref := fmt.Sprintf("10000000-0000-0000-0000-%012d", catalogSeq)
		description, _ := req.Attributes["Наименование"].(string)
		code, _ := req.Attributes["Код"].(string)
		if code == "" {
			code = fmt.Sprintf("%05d", catalogSeq)
		}
		parent, _ := req.Attributes["Родитель"].(string)
		owner, _ := req.Attributes["Владелец"].(string)
		attrs := map[string]any{}
		for k, v := range req.Attributes {
			if k != "Наименование" && k != "Код" && k != "Родитель" && k != "Владелец" {
				attrs[k] = v
			}
		}
		catalogItems[ref] = &mockCatalogItem{
			typ:             req.Type,
			code:            code,
			description:     description,
			isGroup:         req.IsGroup,
			parent:          parent,
			owner:           owner,
			attributes:      attrs,
			tabularSections: req.TabularSections,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"ref": ref, "code": code, "description": description, "is_group": req.IsGroup,
		})
	})

	mux.HandleFunc("/catalog/update", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type            string                      `json:"type"`
			Ref             string                      `json:"ref"`
			Attributes      map[string]any              `json:"attributes"`
			TabularSections map[string][]map[string]any `json:"tabular_sections"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		item, ok := catalogItems[req.Ref]
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Элемент не найден: " + req.Ref})
			return
		}
		if item.attributes == nil {
			item.attributes = map[string]any{}
		}
		for k, v := range req.Attributes {
			switch k {
			case "Наименование":
				item.description, _ = v.(string)
			case "Код":
				item.code, _ = v.(string)
			case "Родитель":
				item.parent, _ = v.(string)
			case "Владелец":
				item.owner, _ = v.(string)
			default:
				item.attributes[k] = v
			}
		}
		if item.tabularSections == nil {
			item.tabularSections = map[string][]map[string]any{}
		}
		for k, v := range req.TabularSections {
			item.tabularSections[k] = v
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"ref": req.Ref, "type": item.typ, "code": item.code, "description": item.description,
			"is_group": item.isGroup, "deletion_mark": item.deletionMark,
			"parent": item.parent, "owner": item.owner,
			"attributes": item.attributes, "tabular_sections": item.tabularSections,
		})
	})

	mux.HandleFunc("/catalog/", func(w http.ResponseWriter, r *http.Request) {
		// GET /catalog/{type}/{ref}
		path := strings.TrimPrefix(r.URL.Path, "/catalog/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ref := parts[1]
		item, ok := catalogItems[ref]
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Элемент не найден: " + ref})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"ref": ref, "type": item.typ, "code": item.code, "description": item.description,
			"is_group": item.isGroup, "deletion_mark": item.deletionMark,
			"parent": item.parent, "owner": item.owner,
			"attributes": item.attributes, "tabular_sections": item.tabularSections,
		})
	})

	mux.HandleFunc("/object/deletion-mark", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ObjectKind string `json:"object_kind"`
			Type       string `json:"type"`
			Ref        string `json:"ref"`
			Mark       bool   `json:"mark"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		switch req.ObjectKind {
		case "Document":
			doc, ok := docs[req.Ref]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Документ не найден: " + req.Ref})
				return
			}
			doc.deletionMark = req.Mark
			// Mirrors real 1C platform behaviour: marking a posted document for
			// deletion automatically unposts it.
			if req.Mark {
				doc.posted = false
			}
			json.NewEncoder(w).Encode(map[string]any{"ref": req.Ref, "deletion_mark": doc.deletionMark, "posted": doc.posted})
		case "Catalog":
			item, ok := catalogItems[req.Ref]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "message": "Элемент не найден: " + req.Ref})
				return
			}
			item.deletionMark = req.Mark
			json.NewEncoder(w).Encode(map[string]any{"ref": req.Ref, "deletion_mark": item.deletionMark})
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "validation_failed", "message": "Unknown object_kind: " + req.ObjectKind})
		}
	})

	mux.HandleFunc("/find", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kind := q.Get("type")
		name := q.Get("name")
		query := strings.ToLower(q.Get("query"))
		includeMarked := q.Get("include_marked") == "true"

		type item struct {
			ref, presentation, code, typeName, date string
			deletionMark, isGroup                   bool
			posted                                  *bool
		}
		var matches []item

		switch kind {
		case "Catalog":
			for ref, ci := range catalogItems {
				if ci.typ != name {
					continue
				}
				if ci.deletionMark && !includeMarked {
					continue
				}
				if query != "" && !strings.Contains(strings.ToLower(ci.description), query) && !strings.Contains(strings.ToLower(ci.code), query) {
					continue
				}
				matches = append(matches, item{
					ref: ref, presentation: ci.description, code: ci.code,
					typeName: "СправочникСсылка." + ci.typ, deletionMark: ci.deletionMark, isGroup: ci.isGroup,
				})
			}
		case "Document":
			for ref, doc := range docs {
				if doc.typ != name {
					continue
				}
				if doc.deletionMark && !includeMarked {
					continue
				}
				if query != "" && !strings.Contains(strings.ToLower(doc.number), query) {
					continue
				}
				posted := doc.posted
				matches = append(matches, item{
					ref: ref, presentation: doc.number, code: doc.number,
					typeName: "ДокументСсылка." + doc.typ, deletionMark: doc.deletionMark,
					date: doc.date, posted: &posted,
				})
			}
		}

		limit := 20
		if l := q.Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		total := len(matches)
		truncated := false
		if len(matches) > limit {
			matches = matches[:limit]
			truncated = true
		}

		resultItems := make([]map[string]any, 0, len(matches))
		for _, m := range matches {
			entry := map[string]any{
				"ref": m.ref, "presentation": m.presentation, "code": m.code,
				"type_name": m.typeName, "deletion_mark": m.deletionMark, "is_group": m.isGroup,
			}
			if m.date != "" {
				entry["date"] = m.date
			}
			if m.posted != nil {
				entry["posted"] = *m.posted
			}
			resultItems = append(resultItems, entry)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"items": resultItems, "total": total, "truncated": truncated,
		})
	})

	mux.HandleFunc("/validate-query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		upper := strings.ToUpper(strings.TrimSpace(req.Query))
		if strings.HasPrefix(upper, "ВЫБРАТЬ") || strings.HasPrefix(upper, "SELECT") {
			json.NewEncoder(w).Encode(map[string]any{"valid": true})
		} else {
			json.NewEncoder(w).Encode(map[string]any{
				"valid":  false,
				"errors": []string{"Ожидается ключевое слово ВЫБРАТЬ"},
			})
		}
	})

	return mux
}

// setupIntegration creates a mock 1C server and connected MCP client session.
// enableWrites controls whether the accounting write tools (create_document,
// post_document, unpost_document) are registered on the server. writeBlacklist
// is passed through to server.New unchanged (nil disables the blacklist check).
func setupIntegration(t *testing.T, enableWrites bool, writeBlacklist []string) (*mcp.ClientSession, func()) {
	t.Helper()

	mock := httptest.NewServer(mock1CHandler())
	client := onec.NewClient(mock.URL, "", "")

	// Create a temp dump directory for search_code tests.
	dumpDir := t.TempDir()
	mkBSL(t, dumpDir, "Documents/РеализацияТоваровУслуг/Ext/ObjectModule.bsl",
		"Процедура ОбработкаПроведения(Отказ, РежимПроведения)\n\t// Код проведения\nКонецПроцедуры\n")
	mkBSL(t, dumpDir, "Documents/ПоступлениеТоваровУслуг/Ext/ObjectModule.bsl",
		"Процедура ОбработкаПроведения(Отказ)\n\t// Проведение поступления\nКонецПроцедуры\n")

	dumpIndex, err := dump.NewIndex(dumpDir, "", false)
	if err != nil {
		mock.Close()
		t.Fatalf("NewIndex: %v", err)
	}

	deadline := time.After(30 * time.Second)
	for !dumpIndex.Ready() {
		select {
		case <-deadline:
			dumpIndex.Close()
			mock.Close()
			t.Fatal("timed out waiting for dump index to become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	var writeClient *onec.Client
	if enableWrites {
		writeClient = client
	}
	srv := New("test", client, dumpIndex, writeClient, writeBlacklist)

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err = srv.Connect(ctx, st, nil)
	if err != nil {
		mock.Close()
		t.Fatalf("server connect: %v", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := mcpClient.Connect(ctx, ct, nil)
	if err != nil {
		mock.Close()
		t.Fatalf("client connect: %v", err)
	}

	cleanup := func() {
		session.Close()
		dumpIndex.Close()
		mock.Close()
	}
	return session, cleanup
}

func TestIntegration_ListTools(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	expected := []string{
		"get_metadata_tree", "get_object_structure", "execute_query",
		"search_code", "get_form_structure", "validate_query",
		"get_event_log", "get_configuration_info", "bsl_syntax_help",
		"get_document", "propose_module_change", "list_proposed_changes",
		"find_ref", "get_catalog_item",
	}
	for _, want := range expected {
		if !toolNames[want] {
			t.Errorf("expected tool %q in list, got: %v", want, toolNames)
		}
	}

	if len(result.Tools) != len(expected) {
		t.Errorf("expected %d tools, got %d: %v", len(expected), len(result.Tools), toolNames)
	}

	for _, absent := range []string{"create_document", "post_document", "unpost_document"} {
		if toolNames[absent] {
			t.Errorf("write tool %q must NOT be listed when enableWrites=false", absent)
		}
	}
}

func TestIntegration_ListTools_WritesEnabled(t *testing.T) {
	session, cleanup := setupIntegration(t, true, nil)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	for _, want := range []string{
		"create_document", "post_document", "unpost_document", "get_document",
		"update_document", "create_catalog_item", "update_catalog_item",
		"set_deletion_mark", "find_ref", "get_catalog_item",
	} {
		if !toolNames[want] {
			t.Errorf("expected write tool %q in list when enableWrites=true, got: %v", want, toolNames)
		}
	}
}

// TestIntegration_DocumentWriteFlow exercises the full create -> get -> post ->
// get cycle through the MCP protocol, mirroring exactly what an LLM client
// would do: create a draft (never posted by create_document itself), verify
// it is unposted, post it explicitly, then verify the posted state.
func TestIntegration_DocumentWriteFlow(t *testing.T) {
	session, cleanup := setupIntegration(t, true, nil)
	defer cleanup()
	ctx := context.Background()

	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_document",
		Arguments: map[string]any{
			"document_type": "РеализацияТоваровУслуг",
			"attributes":    map[string]any{"Организация": "org-guid"},
		},
	})
	if err != nil {
		t.Fatalf("create_document error: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create_document returned tool error: %v", createResult.Content)
	}
	createText := createResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(createText, "НЕ проведён") {
		t.Errorf("expected draft creation text to state the document is not posted, got:\n%s", createText)
	}

	ref := extractRef(t, createText)

	getResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_document",
		Arguments: map[string]any{"document_type": "РеализацияТоваровУслуг", "ref": ref},
	})
	if err != nil {
		t.Fatalf("get_document error: %v", err)
	}
	getText := getResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(getText, "Проведён: false") {
		t.Errorf("expected draft to be unposted, got:\n%s", getText)
	}

	postResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "post_document",
		Arguments: map[string]any{"document_type": "РеализацияТоваровУслуг", "ref": ref},
	})
	if err != nil {
		t.Fatalf("post_document error: %v", err)
	}
	if postResult.IsError {
		t.Fatalf("post_document returned tool error: %v", postResult.Content)
	}

	getResult2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_document",
		Arguments: map[string]any{"document_type": "РеализацияТоваровУслуг", "ref": ref},
	})
	if err != nil {
		t.Fatalf("get_document (after post) error: %v", err)
	}
	getText2 := getResult2.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(getText2, "Проведён: true") {
		t.Errorf("expected document to be posted after post_document, got:\n%s", getText2)
	}
}

// TestIntegration_PostDocument_StructuredFailure verifies that a posting_failed
// error from 1C surfaces as an MCP tool-level error (IsError=true) with the
// exact 1C message, not a generic Go/protocol error — required so the LLM can
// see the reason and explain it or self-correct.
func TestIntegration_PostDocument_StructuredFailure(t *testing.T) {
	session, cleanup := setupIntegration(t, true, nil)
	defer cleanup()
	ctx := context.Background()

	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_document",
		Arguments: map[string]any{
			"document_type": "РеализацияТоваровУслуг",
			"attributes":    map[string]any{"Организация": "FORCE_ERROR"},
		},
	})
	if err != nil {
		t.Fatalf("create_document error: %v", err)
	}
	ref := extractRef(t, createResult.Content[0].(*mcp.TextContent).Text)

	postResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "post_document",
		Arguments: map[string]any{"document_type": "РеализацияТоваровУслуг", "ref": ref},
	})
	if err != nil {
		t.Fatalf("post_document transport error: %v", err)
	}
	if !postResult.IsError {
		t.Fatalf("expected IsError=true for posting_failed, got success: %v", postResult.Content)
	}
	text := postResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "posting_failed") || !strings.Contains(text, "Недостаточно остатков") {
		t.Errorf("expected structured posting_failed message, got:\n%s", text)
	}
}

// extractRef pulls the ref=<value> token out of create_document's response text.
func extractRef(t *testing.T, text string) string {
	t.Helper()
	idx := strings.Index(text, "ref=")
	if idx < 0 {
		t.Fatalf("no ref= found in text:\n%s", text)
	}
	rest := text[idx+len("ref="):]
	end := strings.IndexAny(rest, ",)\n")
	if end < 0 {
		end = len(rest)
	}
	return rest[:end]
}

func TestIntegration_MetadataTree(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	// Without filter -- summary with category names and counts.
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_metadata_tree",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{
		"Справочники", "Документы", "Регистры сведений",
		"Регистры накопления", "Общие модули",
		"Перечисления", "Планы счетов", "Роли",
		"Подсистемы", "HTTP-сервисы", "Планы обмена", "Константы",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in summary, got:\n%s", want, text)
		}
	}

	// With filter -- detailed list of objects in category.
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_metadata_tree",
		Arguments: map[string]any{"filter": "Справочники"},
	})
	if err != nil {
		t.Fatalf("CallTool with filter error: %v", err)
	}
	text = result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"Контрагенты", "Номенклатура"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in filtered response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ObjectStructure(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_object_structure",
		Arguments: map[string]any{
			"object_type": "Document",
			"object_name": "РеализацияТоваровУслуг",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"РеализацияТоваровУслуг", "Контрагент", "СуммаДокумента", "Товары", "Номенклатура"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ObjectStructure_Register(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_object_structure",
		Arguments: map[string]any{
			"object_type": "AccumulationRegister",
			"object_name": "ТоварыНаСкладах",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ТоварыНаСкладах", "Измерения", "Номенклатура", "Склад", "Ресурсы", "Количество"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ObjectStructure_NotFound(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_object_structure",
		Arguments: map[string]any{
			"object_type": "Document",
			"object_name": "НесуществующийДокумент",
		},
	})
	if err == nil {
		t.Fatal("expected error for non-existent object")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}

func TestIntegration_FormStructure(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_form_structure",
		Arguments: map[string]any{
			"object_type": "Document",
			"object_name": "РеализацияТоваровУслуг",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ФормаДокумента", "Контрагент", "ПолеВвода", "Провести", "ПриОткрытии"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ConfigInfo(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_configuration_info",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{
		"БухгалтерияПредприятия",
		"3.0.150.1",
		"8.3.25.1000",
		"Файловый",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_SearchCode(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "search_code",
		Arguments: map[string]any{
			"query": "ОбработкаПроведения",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{
		"ОбработкаПроведения",
		"Документ.РеализацияТоваровУслуг.МодульОбъекта",
		"Документ.ПоступлениеТоваровУслуг.МодульОбъекта",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ProposeModuleChange(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()
	ctx := context.Background()

	const moduleID = "Документ.РеализацияТоваровУслуг.МодульОбъекта"
	const newContent = "Процедура ОбработкаПроведения(Отказ, РежимПроведения)\n\t// Новый код проведения\nКонецПроцедуры\n"

	proposeRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "propose_module_change",
		Arguments: map[string]any{
			"module_id":   moduleID,
			"new_content": newContent,
			"rationale":   "test proposal - must not be applied automatically",
		},
	})
	if err != nil {
		t.Fatalf("propose_module_change error: %v", err)
	}
	if proposeRes.IsError {
		t.Fatalf("propose_module_change returned tool error: %v", proposeRes.Content)
	}
	text := proposeRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "НЕ применено") {
		t.Errorf("expected response to state the change was not applied, got:\n%s", text)
	}
	if !strings.Contains(text, "Новый код проведения") {
		t.Errorf("expected diff to show the new content, got:\n%s", text)
	}

	// The underlying dump file must be untouched by propose_module_change:
	// searching for the original fixture text should still find it.
	searchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_code",
		Arguments: map[string]any{"query": "Код проведения"},
	})
	if err != nil {
		t.Fatalf("search_code (post-propose) error: %v", err)
	}
	searchText := searchRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(searchText, "Документ.РеализацияТоваровУслуг.МодульОбъекта") {
		t.Errorf("expected original module content to be unchanged on disk, got search result:\n%s", searchText)
	}

	listRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_proposed_changes",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("list_proposed_changes error: %v", err)
	}
	listText := listRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(listText, moduleID) {
		t.Errorf("expected proposed change to be listed, got:\n%s", listText)
	}
}

func TestIntegration_BSLSyntaxHelp(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bsl_syntax_help",
		Arguments: map[string]any{
			"query": "СтрНайти",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "СтрНайти") {
		t.Errorf("expected СтрНайти in response, got:\n%s", text)
	}
	if !strings.Contains(text, "StrFind") {
		t.Errorf("expected StrFind in response, got:\n%s", text)
	}
}

func TestIntegration_ExecuteQuery(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "execute_query",
		Arguments: map[string]any{
			"query": "ВЫБРАТЬ Наименование ИЗ Справочник.Контрагенты",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"Наименование", "ООО Ромашка", "ИП Иванов"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ValidateQuery_Valid(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "validate_query",
		Arguments: map[string]any{
			"query": "ВЫБРАТЬ Наименование ИЗ Справочник.Контрагенты",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "корректен") {
		t.Errorf("expected 'корректен' in response for valid query, got:\n%s", text)
	}
}

func TestIntegration_ValidateQuery_Invalid(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "validate_query",
		Arguments: map[string]any{
			"query": "ОБНОВИТЬ Справочник.Контрагенты",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "ошибки") {
		t.Errorf("expected 'ошибки' in response for invalid query, got:\n%s", text)
	}
}

func TestIntegration_EventLog(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_event_log",
		Arguments: map[string]any{
			"level": "Ошибка",
			"limit": 10,
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{
		"Журнал регистрации",
		"Ошибка",
		"Администратор",
		"РеализацияТоваровУслуг",
		"Информация",
		"Бухгалтер",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response, got:\n%s", want, text)
		}
	}
}

func TestIntegration_ListPrompts(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListPrompts error: %v", err)
	}

	expected := map[string]bool{
		"review_module":            false,
		"write_posting":            false,
		"optimize_query":           false,
		"explain_config":           false,
		"analyze_error":            false,
		"find_duplicates":          false,
		"write_report":             false,
		"explain_object":           false,
		"1c_query_syntax":          false,
		"1c_metadata_navigation":   false,
		"1c_development_workflow":  false,
	}

	for _, p := range result.Prompts {
		if _, ok := expected[p.Name]; ok {
			expected[p.Name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected prompt %q not found in ListPrompts result", name)
		}
	}

	if len(result.Prompts) != len(expected) {
		t.Errorf("expected %d prompts, got %d", len(expected), len(result.Prompts))
	}
}

func TestIntegration_GetPrompt_ReviewModule(t *testing.T) {
	session, cleanup := setupIntegration(t, false, nil)
	defer cleanup()

	result, err := session.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name: "review_module",
		Arguments: map[string]string{
			"object_type": "Document",
			"object_name": "РеализацияТоваровУслуг",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt error: %v", err)
	}

	if result.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}

	msg := result.Messages[0]
	if msg.Role != "user" {
		t.Errorf("expected role \"user\", got %q", msg.Role)
	}

	tc, ok := msg.Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", msg.Content)
	}

	for _, keyword := range []string{
		"Document",
		"РеализацияТоваровУслуг",
		"get_object_structure",
		"search_code",
	} {
		if !strings.Contains(tc.Text, keyword) {
			t.Errorf("expected %q in prompt text, got:\n%s", keyword, tc.Text)
		}
	}
}

// TestIntegrationFindRef exercises find_ref end to end: an empty result before
// anything exists, then a match once a catalog item has been created via
// create_catalog_item, mirroring how an LLM resolves a GUID before referencing
// it in another write call.
func TestIntegrationFindRef(t *testing.T) {
	session, cleanup := setupIntegration(t, true, nil)
	defer cleanup()
	ctx := context.Background()

	emptyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "find_ref",
		Arguments: map[string]any{
			"object_kind": "Catalog",
			"name":        "Контрагенты",
			"query":       "Ромашка",
		},
	})
	if err != nil {
		t.Fatalf("find_ref error: %v", err)
	}
	emptyText := emptyResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(emptyText, "Ничего не найдено") {
		t.Errorf("expected no matches before creation, got:\n%s", emptyText)
	}

	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_catalog_item",
		Arguments: map[string]any{
			"catalog_type": "Контрагенты",
			"attributes":   map[string]any{"Наименование": "ООО Ромашка"},
		},
	})
	if err != nil {
		t.Fatalf("create_catalog_item error: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create_catalog_item returned tool error: %v", createResult.Content)
	}
	ref := extractRef(t, createResult.Content[0].(*mcp.TextContent).Text)

	foundResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "find_ref",
		Arguments: map[string]any{
			"object_kind": "Catalog",
			"name":        "Контрагенты",
			"query":       "Ромашка",
		},
	})
	if err != nil {
		t.Fatalf("find_ref (after create) error: %v", err)
	}
	foundText := foundResult.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ООО Ромашка", ref} {
		if !strings.Contains(foundText, want) {
			t.Errorf("expected %q in find_ref result, got:\n%s", want, foundText)
		}
	}
}

// TestIntegrationCatalogLifecycle drives create -> get -> update -> deletion
// mark for a catalog item through the MCP protocol, the catalog counterpart of
// TestIntegration_DocumentWriteFlow.
func TestIntegrationCatalogLifecycle(t *testing.T) {
	session, cleanup := setupIntegration(t, true, nil)
	defer cleanup()
	ctx := context.Background()

	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_catalog_item",
		Arguments: map[string]any{
			"catalog_type": "Контрагенты",
			"attributes":   map[string]any{"Наименование": "ООО Ромашка", "ИНН": "301234567"},
		},
	})
	if err != nil {
		t.Fatalf("create_catalog_item error: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create_catalog_item returned tool error: %v", createResult.Content)
	}
	ref := extractRef(t, createResult.Content[0].(*mcp.TextContent).Text)

	getResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_catalog_item",
		Arguments: map[string]any{"catalog_type": "Контрагенты", "ref": ref},
	})
	if err != nil {
		t.Fatalf("get_catalog_item error: %v", err)
	}
	getText := getResult.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ООО Ромашка", "301234567", "Пометка удаления: false"} {
		if !strings.Contains(getText, want) {
			t.Errorf("expected %q in get_catalog_item result, got:\n%s", want, getText)
		}
	}

	updateResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_catalog_item",
		Arguments: map[string]any{
			"catalog_type": "Контрагенты",
			"ref":          ref,
			"attributes":   map[string]any{"Наименование": "ООО Ромашка Новая"},
		},
	})
	if err != nil {
		t.Fatalf("update_catalog_item error: %v", err)
	}
	if updateResult.IsError {
		t.Fatalf("update_catalog_item returned tool error: %v", updateResult.Content)
	}
	updateText := updateResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(updateText, "ООО Ромашка Новая") {
		t.Errorf("expected updated name in update_catalog_item result, got:\n%s", updateText)
	}

	markResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "set_deletion_mark",
		Arguments: map[string]any{
			"object_kind": "Catalog",
			"type":        "Контрагенты",
			"ref":         ref,
		},
	})
	if err != nil {
		t.Fatalf("set_deletion_mark error: %v", err)
	}
	if markResult.IsError {
		t.Fatalf("set_deletion_mark returned tool error: %v", markResult.Content)
	}

	getResult2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_catalog_item",
		Arguments: map[string]any{"catalog_type": "Контрагенты", "ref": ref},
	})
	if err != nil {
		t.Fatalf("get_catalog_item (after mark) error: %v", err)
	}
	getText2 := getResult2.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(getText2, "Пометка удаления: true") {
		t.Errorf("expected deletion mark set, got:\n%s", getText2)
	}
}

// TestIntegrationUpdateDocument verifies update_document merges attributes
// without touching posted state, and that posting a document and then marking
// it for deletion auto-unposts it (mirrors the real 1C platform behaviour
// documented in set_deletion_mark's description).
func TestIntegrationUpdateDocument(t *testing.T) {
	session, cleanup := setupIntegration(t, true, nil)
	defer cleanup()
	ctx := context.Background()

	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_document",
		Arguments: map[string]any{
			"document_type": "РеализацияТоваровУслуг",
			"attributes":    map[string]any{"Организация": "org-guid"},
		},
	})
	if err != nil {
		t.Fatalf("create_document error: %v", err)
	}
	ref := extractRef(t, createResult.Content[0].(*mcp.TextContent).Text)

	updateResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_document",
		Arguments: map[string]any{
			"document_type": "РеализацияТоваровУслуг",
			"ref":           ref,
			"attributes":    map[string]any{"Комментарий": "изменено через MCP"},
		},
	})
	if err != nil {
		t.Fatalf("update_document error: %v", err)
	}
	if updateResult.IsError {
		t.Fatalf("update_document returned tool error: %v", updateResult.Content)
	}
	updateText := updateResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(updateText, "изменено через MCP") {
		t.Errorf("expected updated comment in update_document result, got:\n%s", updateText)
	}

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "post_document",
		Arguments: map[string]any{"document_type": "РеализацияТоваровУслуг", "ref": ref},
	}); err != nil {
		t.Fatalf("post_document error: %v", err)
	}

	markResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "set_deletion_mark",
		Arguments: map[string]any{
			"object_kind": "Document",
			"type":        "РеализацияТоваровУслуг",
			"ref":         ref,
		},
	})
	if err != nil {
		t.Fatalf("set_deletion_mark error: %v", err)
	}
	markText := markResult.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(markText, "Текущее состояние проведения: false") {
		t.Errorf("expected posting a marked document to be auto-unposted, got:\n%s", markText)
	}
}

// TestIntegrationBlacklistedDocumentType verifies that a document type
// matching the write blacklist is rejected by every write tool before any
// mock 1C HTTP call is made, regardless of the underlying (mock) user's
// rights — the safety layer tools.IsWriteBlacklisted adds on top of
// writeClient permissions.
func TestIntegrationBlacklistedDocumentType(t *testing.T) {
	session, cleanup := setupIntegration(t, true, tools.DefaultWriteTypeBlacklist)
	defer cleanup()
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_document",
		Arguments: map[string]any{
			"document_type": "dibank_ПлатежноеПоручениеИсходящее",
			"attributes":    map[string]any{"Организация": "org-guid"},
		},
	})
	if err != nil {
		t.Fatalf("create_document transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for blacklisted document type")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "dibank_ПлатежноеПоручениеИсходящее") {
		t.Errorf("expected blacklisted type name in error text, got:\n%s", text)
	}
}

func mkBSL(t *testing.T, base, relPath, content string) {
	t.Helper()
	full := filepath.Join(base, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
