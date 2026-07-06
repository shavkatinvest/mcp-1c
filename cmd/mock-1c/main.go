package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/feenlace/mcp-1c/onec"
)

// objectKey combines type and name for map lookup.
type objectKey struct {
	typ  string
	name string
}

var (
	metadata = map[string][]string{
		"Справочники": {
			"Контрагенты",
			"Номенклатура",
			"Организации",
			"Сотрудники",
			"Валюты",
			"Склады",
			"БанковскиеСчета",
			"ДоговорыКонтрагентов",
			"ЕдиницыИзмерения",
		},
		"Документы": {
			"РеализацияТоваровУслуг",
			"ПоступлениеТоваровУслуг",
			"СчетНаОплатуПокупателю",
			"ПлатежноеПоручение",
			"КассовыйОрдер",
			"АвансовыйОтчет",
			"ОперацияБух",
		},
		"Перечисления": {
			"СтавкиНДС",
			"ВидыНоменклатуры",
			"ВидыОпераций",
			"ТипыКонтактнойИнформации",
		},
		"Обработки": {
			"ЗагрузкаДанныхИзФайла",
			"ГрупповоеИзменениеОбъектов",
		},
		"Отчеты": {
			"ОборотноСальдоваяВедомость",
			"КарточкаСчета",
			"АнализСубконто",
		},
		"РегистрыСведений": {
			"КурсыВалют",
			"АдресныйКлассификатор",
			"НастройкиУчетнойПолитики",
		},
		"РегистрыНакопления": {
			"ТоварыНаСкладах",
			"ВзаиморасчетыСКонтрагентами",
		},
		"РегистрыБухгалтерии": {
			"Хозрасчетный",
		},
		"РегистрыРасчета":          {},
		"ПланыСчетов":              {"Хозрасчетный"},
		"ПланыВидовХарактеристик":  {"ВидыСубконтоХозрасчетные"},
		"ПланыВидовРасчета":        {},
		"ПланыОбмена":              {"ОбменБухгалтерия"},
		"БизнесПроцессы":           {},
		"Задачи":                   {},
		"ЖурналыДокументов":        {"ЖурналОпераций"},
		"Константы":                {"ВалютаРегламентированногоУчета", "ОсновнаяОрганизация"},
		"ОбщиеМодули": {
			"ОбщегоНазначения",
			"ОбщегоНазначенияКлиентСервер",
			"УправлениеПечатью",
		},
		"ОбщиеФормы":        {"ФормаВопроса", "ФормаПредупреждения"},
		"ОбщиеКоманды":      {},
		"ОбщиеМакеты":       {"МакетПечати"},
		"Роли":              {"Администратор", "Бухгалтер", "ТолькоПросмотр"},
		"Подсистемы":        {"Бухгалтерия", "Зарплата", "Администрирование"},
		"РегулярныеЗадания": {"ОбновлениеКурсовВалют"},
		"ВебСервисы":        {},
		"HTTPСервисы":       {"MCPService"},
	}

	objects = map[objectKey]onec.ObjectStructure{
		{typ: "Document", name: "РеализацияТоваровУслуг"}: {
			Name:    "РеализацияТоваровУслуг",
			Synonym: "Реализация (акты, накладные, УПД)",
			Attributes: []onec.Attribute{
				{Name: "Контрагент", Synonym: "Контрагент", Type: "СправочникСсылка.Контрагенты"},
				{Name: "Организация", Synonym: "Организация", Type: "СправочникСсылка.Организации"},
				{Name: "Склад", Synonym: "Склад", Type: "СправочникСсылка.Склады"},
				{Name: "Валюта", Synonym: "Валюта расчётов", Type: "СправочникСсылка.Валюты"},
				{Name: "ДоговорКонтрагента", Synonym: "Договор", Type: "СправочникСсылка.ДоговорыКонтрагентов"},
				{Name: "СуммаДокумента", Synonym: "Сумма", Type: "Число"},
				{Name: "Комментарий", Synonym: "Комментарий", Type: "Строка"},
			},
			TabularParts: []onec.TabularPart{
				{
					Name: "Товары",
					Attributes: []onec.Attribute{
						{Name: "Номенклатура", Synonym: "Номенклатура", Type: "СправочникСсылка.Номенклатура"},
						{Name: "Количество", Synonym: "Количество", Type: "Число"},
						{Name: "Цена", Synonym: "Цена", Type: "Число"},
						{Name: "Сумма", Synonym: "Сумма", Type: "Число"},
						{Name: "СтавкаНДС", Synonym: "Ставка НДС", Type: "ПеречислениеСсылка.СтавкиНДС"},
						{Name: "СуммаНДС", Synonym: "Сумма НДС", Type: "Число"},
					},
				},
				{
					Name: "Услуги",
					Attributes: []onec.Attribute{
						{Name: "Номенклатура", Synonym: "Номенклатура", Type: "СправочникСсылка.Номенклатура"},
						{Name: "Количество", Synonym: "Количество", Type: "Число"},
						{Name: "Цена", Synonym: "Цена", Type: "Число"},
						{Name: "Сумма", Synonym: "Сумма", Type: "Число"},
						{Name: "СодержаниеУслуги", Synonym: "Содержание", Type: "Строка"},
					},
				},
			},
		},
		{typ: "Catalog", name: "Контрагенты"}: {
			Name:    "Контрагенты",
			Synonym: "Контрагенты",
			Attributes: []onec.Attribute{
				{Name: "ИНН", Synonym: "ИНН", Type: "Строка"},
				{Name: "КПП", Synonym: "КПП", Type: "Строка"},
				{Name: "НаименованиеПолное", Synonym: "Полное наименование", Type: "Строка"},
				{Name: "ЮридическийАдрес", Synonym: "Юридический адрес", Type: "Строка"},
				{Name: "ОсновнойДоговор", Synonym: "Основной договор", Type: "СправочникСсылка.ДоговорыКонтрагентов"},
				{Name: "ОсновнойБанковскийСчет", Synonym: "Основной банковский счёт", Type: "СправочникСсылка.БанковскиеСчета"},
			},
			TabularParts: []onec.TabularPart{
				{
					Name: "КонтактнаяИнформация",
					Attributes: []onec.Attribute{
						{Name: "Тип", Synonym: "Тип", Type: "ПеречислениеСсылка.ТипыКонтактнойИнформации"},
						{Name: "Представление", Synonym: "Представление", Type: "Строка"},
					},
				},
			},
		},
		{typ: "Catalog", name: "Номенклатура"}: {
			Name:    "Номенклатура",
			Synonym: "Номенклатура",
			Attributes: []onec.Attribute{
				{Name: "Артикул", Synonym: "Артикул", Type: "Строка"},
				{Name: "ЕдиницаИзмерения", Synonym: "Единица измерения", Type: "СправочникСсылка.ЕдиницыИзмерения"},
				{Name: "ВидНоменклатуры", Synonym: "Вид номенклатуры", Type: "ПеречислениеСсылка.ВидыНоменклатуры"},
				{Name: "СтавкаНДС", Synonym: "Ставка НДС", Type: "ПеречислениеСсылка.СтавкиНДС"},
				{Name: "Описание", Synonym: "Описание", Type: "Строка"},
			},
		},
		{typ: "AccumulationRegister", name: "ТоварыНаСкладах"}: {
			Name:    "ТоварыНаСкладах",
			Synonym: "Товары на складах",
			Dimensions: []onec.Attribute{
				{Name: "Номенклатура", Synonym: "Номенклатура", Type: "СправочникСсылка.Номенклатура"},
				{Name: "Склад", Synonym: "Склад", Type: "СправочникСсылка.Склады"},
			},
			Resources: []onec.Attribute{
				{Name: "Количество", Synonym: "Количество", Type: "Число"},
			},
			Attributes: []onec.Attribute{},
		},
	}

)

// mockDocument is an in-memory document record used by the write endpoints
// (/document, /document/post, /document/unpost, /document/{type}/{ref}).
// It exists purely to let tools/create_document.go and friends be exercised
// locally without a real 1C instance.
type mockDocument struct {
	Type            string
	Number          string
	Date            string
	Posted          bool
	DeletionMark    bool
	Attributes      map[string]any
	TabularSections map[string][]map[string]any
}

// mockCatalogItem is an in-memory catalog item record used by the catalog
// write endpoints (/catalog, /catalog/update, /catalog/{type}/{ref}).
type mockCatalogItem struct {
	Type            string
	Code            string
	Description     string
	IsGroup         bool
	DeletionMark    bool
	Parent          string
	Owner           string
	Attributes      map[string]any
	TabularSections map[string][]map[string]any
}

var (
	documents   = map[string]*mockDocument{}
	documentSeq = 0

	catalogItems = map[string]*mockCatalogItem{}
	catalogSeq   = 0
)

// forceErrorOrgValue is a magic Организация attribute value that makes
// /document/post return a structured posting_failed error, so the error path
// of post_document can be exercised without a real 1C posting-rule violation.
const forceErrorOrgValue = "FORCE_ERROR"

func newDocumentRef() string {
	documentSeq++
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", documentSeq)
}

func handleDocumentCreate(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Type            string                      `json:"type"`
		Attributes      map[string]any              `json:"attributes"`
		TabularSections map[string][]map[string]any `json:"tabular_sections"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}
	if req.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validation_failed", "message": "type is required"})
		return
	}

	ref := newDocumentRef()
	doc := &mockDocument{
		Type:            req.Type,
		Number:          fmt.Sprintf("ЗН-%06d", documentSeq),
		Date:            "2026-07-02T10:00:00",
		Posted:          false,
		Attributes:      req.Attributes,
		TabularSections: req.TabularSections,
	}
	documents[ref] = doc

	writeJSON(w, http.StatusOK, map[string]any{
		"ref": ref, "number": doc.Number, "date": doc.Date, "posted": false,
	})
}

func handleDocumentGet(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

	path := strings.TrimPrefix(r.URL.Path, "/mcp/document/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "validation_failed", "message": "Invalid path. Expected /mcp/document/{type}/{ref}",
		})
		return
	}
	ref := parts[1]

	doc, ok := documents[ref]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Документ не найден: " + ref})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref": ref, "type": doc.Type, "number": doc.Number, "date": doc.Date,
		"posted": doc.Posted, "deletion_mark": doc.DeletionMark,
		"attributes": doc.Attributes, "tabular_sections": doc.TabularSections,
	})
}

func handleDocumentUpdate(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Type            string                      `json:"type"`
		Ref             string                      `json:"ref"`
		Attributes      map[string]any              `json:"attributes"`
		TabularSections map[string][]map[string]any `json:"tabular_sections"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	doc, ok := documents[req.Ref]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Документ не найден: " + req.Ref})
		return
	}
	if doc.Attributes == nil {
		doc.Attributes = map[string]any{}
	}
	for k, v := range req.Attributes {
		doc.Attributes[k] = v
	}
	if doc.TabularSections == nil {
		doc.TabularSections = map[string][]map[string]any{}
	}
	for k, v := range req.TabularSections {
		doc.TabularSections[k] = v
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref": req.Ref, "type": doc.Type, "number": doc.Number, "date": doc.Date,
		"posted": doc.Posted, "deletion_mark": doc.DeletionMark,
		"attributes": doc.Attributes, "tabular_sections": doc.TabularSections,
	})
}

func newCatalogRef() string {
	catalogSeq++
	return fmt.Sprintf("10000000-0000-0000-0000-%012d", catalogSeq)
}

func handleCatalogCreate(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Type            string                      `json:"type"`
		IsGroup         bool                        `json:"is_group"`
		OwnerType       string                      `json:"owner_type"`
		Attributes      map[string]any              `json:"attributes"`
		TabularSections map[string][]map[string]any `json:"tabular_sections"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}
	if req.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validation_failed", "message": "type is required"})
		return
	}

	ref := newCatalogRef()
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
		Type:            req.Type,
		Code:            code,
		Description:     description,
		IsGroup:         req.IsGroup,
		Parent:          parent,
		Owner:           owner,
		Attributes:      attrs,
		TabularSections: req.TabularSections,
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref": ref, "code": code, "description": description, "is_group": req.IsGroup,
	})
}

func handleCatalogUpdate(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Type            string                      `json:"type"`
		Ref             string                      `json:"ref"`
		Attributes      map[string]any              `json:"attributes"`
		TabularSections map[string][]map[string]any `json:"tabular_sections"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	item, ok := catalogItems[req.Ref]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Элемент не найден: " + req.Ref})
		return
	}
	if item.Attributes == nil {
		item.Attributes = map[string]any{}
	}
	for k, v := range req.Attributes {
		switch k {
		case "Наименование":
			item.Description, _ = v.(string)
		case "Код":
			item.Code, _ = v.(string)
		case "Родитель":
			item.Parent, _ = v.(string)
		case "Владелец":
			item.Owner, _ = v.(string)
		default:
			item.Attributes[k] = v
		}
	}
	if item.TabularSections == nil {
		item.TabularSections = map[string][]map[string]any{}
	}
	for k, v := range req.TabularSections {
		item.TabularSections[k] = v
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref": req.Ref, "type": item.Type, "code": item.Code, "description": item.Description,
		"is_group": item.IsGroup, "deletion_mark": item.DeletionMark,
		"parent": item.Parent, "owner": item.Owner,
		"attributes": item.Attributes, "tabular_sections": item.TabularSections,
	})
}

func handleCatalogGet(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

	path := strings.TrimPrefix(r.URL.Path, "/mcp/catalog/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "validation_failed", "message": "Invalid path. Expected /mcp/catalog/{type}/{ref}",
		})
		return
	}
	ref := parts[1]

	item, ok := catalogItems[ref]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Элемент не найден: " + ref})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref": ref, "type": item.Type, "code": item.Code, "description": item.Description,
		"is_group": item.IsGroup, "deletion_mark": item.DeletionMark,
		"parent": item.Parent, "owner": item.Owner,
		"attributes": item.Attributes, "tabular_sections": item.TabularSections,
	})
}

func handleDeletionMark(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		ObjectKind string `json:"object_kind"`
		Type       string `json:"type"`
		Ref        string `json:"ref"`
		Mark       bool   `json:"mark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	switch req.ObjectKind {
	case "Document":
		doc, ok := documents[req.Ref]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Документ не найден: " + req.Ref})
			return
		}
		doc.DeletionMark = req.Mark
		// Mirrors real 1C platform behaviour: marking a posted document for
		// deletion automatically unposts it.
		if req.Mark {
			doc.Posted = false
		}
		writeJSON(w, http.StatusOK, map[string]any{"ref": req.Ref, "deletion_mark": doc.DeletionMark, "posted": doc.Posted})
	case "Catalog":
		item, ok := catalogItems[req.Ref]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Элемент не найден: " + req.Ref})
			return
		}
		item.DeletionMark = req.Mark
		writeJSON(w, http.StatusOK, map[string]any{"ref": req.Ref, "deletion_mark": item.DeletionMark})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validation_failed", "message": "Unknown object_kind: " + req.ObjectKind})
	}
}

func handleFind(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

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
			if ci.Type != name {
				continue
			}
			if ci.DeletionMark && !includeMarked {
				continue
			}
			if query != "" && !strings.Contains(strings.ToLower(ci.Description), query) && !strings.Contains(strings.ToLower(ci.Code), query) {
				continue
			}
			matches = append(matches, item{
				ref: ref, presentation: ci.Description, code: ci.Code,
				typeName: "СправочникСсылка." + ci.Type, deletionMark: ci.DeletionMark, isGroup: ci.IsGroup,
			})
		}
	case "Document":
		for ref, doc := range documents {
			if doc.Type != name {
				continue
			}
			if doc.DeletionMark && !includeMarked {
				continue
			}
			if query != "" && !strings.Contains(strings.ToLower(doc.Number), query) {
				continue
			}
			posted := doc.Posted
			matches = append(matches, item{
				ref: ref, presentation: doc.Number, code: doc.Number,
				typeName: "ДокументСсылка." + doc.Type, deletionMark: doc.DeletionMark,
				date: doc.Date, posted: &posted,
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

	writeJSON(w, http.StatusOK, map[string]any{
		"items": resultItems, "total": total, "truncated": truncated,
	})
}

func handleDocumentPost(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Type string `json:"type"`
		Ref  string `json:"ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	doc, ok := documents[req.Ref]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Документ не найден: " + req.Ref})
		return
	}

	if org, _ := doc.Attributes["Организация"].(string); org == forceErrorOrgValue {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "posting_failed", "message": "Недостаточно остатков на складе",
		})
		return
	}

	doc.Posted = true
	writeJSON(w, http.StatusOK, map[string]any{"ref": req.Ref, "posted": true, "date": doc.Date})
}

func handleDocumentUnpost(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Type string `json:"type"`
		Ref  string `json:"ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	doc, ok := documents[req.Ref]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Документ не найден: " + req.Ref})
		return
	}

	doc.Posted = false
	writeJSON(w, http.StatusOK, map[string]any{"ref": req.Ref, "posted": false, "date": doc.Date})
}

// isSelectQuery checks if a query starts with SELECT/ВЫБРАТЬ keyword.
func isSelectQuery(query string) bool {
	upper := strings.ToUpper(strings.TrimSpace(query))
	return strings.HasPrefix(upper, "ВЫБРАТЬ") || strings.HasPrefix(upper, "SELECT")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func handleMetadata(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	writeJSON(w, http.StatusOK, metadata)
}

func handleObject(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

	// Parse path: /mcp/object/{type}/{name}
	path := strings.TrimPrefix(r.URL.Path, "/mcp/object/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid path. Expected /mcp/object/{type}/{name}",
		})
		return
	}

	key := objectKey{typ: parts[0], name: parts[1]}
	obj, ok := objects[key]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Object not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, obj)
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body"})
		return
	}

	if !isSelectQuery(req.Query) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Only SELECT queries allowed"})
		return
	}

	result := map[string]any{
		"columns":   []string{"Наименование", "ИНН"},
		"rows":      [][]string{{"ООО Ромашка", "7701234567"}, {"ИП Петров", "772987654321"}},
		"total":     2,
		"truncated": false,
	}
	writeJSON(w, http.StatusOK, result)
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

	writeJSON(w, http.StatusOK, map[string]any{
		"name":  "ФормаДокумента",
		"title": "Реализация товаров и услуг",
		"elements": []map[string]any{
			{"name": "Контрагент", "type": "ПолеВвода", "title": "Контрагент", "dataPath": "Объект.Контрагент"},
			{"name": "Организация", "type": "ПолеВвода", "title": "Организация", "dataPath": "Объект.Организация"},
			{"name": "СуммаДокумента", "type": "ПолеВвода", "title": "Сумма", "dataPath": "Объект.СуммаДокумента"},
			{"name": "ТаблицаТоваров", "type": "ТаблицаФормы", "title": "Товары", "dataPath": "Объект.Товары"},
		},
		"commands": []map[string]any{
			{"name": "ПровестиИЗакрыть", "action": "ПровестиИЗакрыть"},
			{"name": "Записать", "action": "Записать"},
		},
		"handlers": []map[string]any{
			{"event": "ПриОткрытии", "handler": "ПриОткрытии"},
			{"event": "ПередЗаписью", "handler": "ПередЗаписью"},
		},
	})
}

func handleValidateQuery(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body"})
		return
	}

	if isSelectQuery(req.Query) {
		writeJSON(w, http.StatusOK, map[string]any{"valid": true})
	} else {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid":  false,
			"errors": []string{"Ожидается ключевое слово ВЫБРАТЬ или SELECT"},
		})
	}
}

func handleEventLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req struct {
		Level string `json:"level"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body"})
		return
	}

	events := []map[string]any{
		{
			"date":     "2026-03-07T14:30:00",
			"level":    "Ошибка",
			"event":    "Данные.Запись",
			"user":     "Администратор",
			"metadata": "Документ.РеализацияТоваровУслуг",
			"comment":  "Ошибка при записи: поле Контрагент не заполнено",
		},
		{
			"date":     "2026-03-07T14:25:00",
			"level":    "Предупреждение",
			"event":    "Данные.Проведение",
			"user":     "Бухгалтер",
			"metadata": "Документ.ПоступлениеТоваровУслуг",
			"comment":  "Отрицательный остаток по регистру ТоварыНаСкладах",
		},
		{
			"date":  "2026-03-07T14:00:00",
			"level": "Информация",
			"event": "Сеанс.Начало",
			"user":  "Администратор",
		},
	}

	if req.Level != "" {
		var filtered []map[string]any
		for _, e := range events {
			if e["level"] == req.Level {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	total := len(events)
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit < total {
		events = events[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"total":  total,
	})
}

func handleConfiguration(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	writeJSON(w, http.StatusOK, map[string]any{
		"name":             "БухгалтерияПредприятия",
		"version":          "3.0.150.28",
		"vendor":           "Фирма \"1С\"",
		"platform_version": "8.3.25.1394",
		"mode":             "file",
	})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	writeJSON(w, http.StatusOK, map[string]string{"version": "0.3.0"})
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	logger := log.New(os.Stderr, "", log.LstdFlags)
	log.SetOutput(os.Stderr)

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp/metadata", handleMetadata)
	mux.HandleFunc("/mcp/object/", handleObject)
	mux.HandleFunc("/mcp/query", handleQuery)

	mux.HandleFunc("/mcp/form/", handleForm)
	mux.HandleFunc("/mcp/validate-query", handleValidateQuery)
	mux.HandleFunc("/mcp/eventlog", handleEventLog)
	mux.HandleFunc("/mcp/configuration", handleConfiguration)
	mux.HandleFunc("/mcp/version", handleVersion)
	mux.HandleFunc("/mcp/document", handleDocumentCreate)
	mux.HandleFunc("/mcp/document/post", handleDocumentPost)
	mux.HandleFunc("/mcp/document/unpost", handleDocumentUnpost)
	mux.HandleFunc("/mcp/document/update", handleDocumentUpdate)
	mux.HandleFunc("/mcp/document/", handleDocumentGet)
	mux.HandleFunc("/mcp/catalog", handleCatalogCreate)
	mux.HandleFunc("/mcp/catalog/update", handleCatalogUpdate)
	mux.HandleFunc("/mcp/catalog/", handleCatalogGet)
	mux.HandleFunc("/mcp/object/deletion-mark", handleDeletionMark)
	mux.HandleFunc("/mcp/find", handleFind)

	addr := fmt.Sprintf(":%d", *port)
	logger.Printf("Mock 1C server listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}
