package model

import "strings"

// Connection is a provider connection of an account.
type Connection struct {
	ConnectionID         string            `json:"connection_id"`
	MerchantConnectionID string            `json:"merchant_connection_id"`
	ProviderID           string            `json:"provider_id"`
	Status               string            `json:"status"`
	FlowType             string            `json:"flow_type"`
	PaymentMethods       []string          `json:"payment_methods,omitempty"`
	Params               []ConnectionParam `json:"params,omitempty"`
	Costs                []map[string]any  `json:"costs,omitempty"`
	CreatedAt            string            `json:"created_at,omitempty"`
	UpdatedAt            string            `json:"updated_at,omitempty"`
}

// ConnectionParam is one provider credential or setting of a connection.
type ConnectionParam struct {
	ParamID string `json:"param_id"`
	Value   string `json:"value,omitempty"`
}

// ConnectionView is the flat table row of a connection.
type ConnectionView struct {
	ConnectionID         string `json:"connection_id"`
	MerchantConnectionID string `json:"merchant_connection_id"`
	ProviderID           string `json:"provider_id"`
	Status               string `json:"status"`
	FlowType             string `json:"flow_type"`
	PaymentMethods       string `json:"payment_methods"`
}

// View flattens a connection into a table row.
func (c *Connection) View() ConnectionView {
	return ConnectionView{
		ConnectionID:         c.ConnectionID,
		MerchantConnectionID: c.MerchantConnectionID,
		ProviderID:           c.ProviderID,
		Status:               c.Status,
		FlowType:             c.FlowType,
		PaymentMethods:       strings.Join(c.PaymentMethods, ","),
	}
}

// ProviderCatalog describes what a provider supports and which parameters a
// connection to it needs.
type ProviderCatalog struct {
	PaymentMethodType []string       `json:"payment_method_type,omitempty"`
	Params            []CatalogParam `json:"params,omitempty"`
}

// CatalogParam is one parameter a provider connection accepts.
type CatalogParam struct {
	ParamID       string `json:"param_id"`
	FieldType     string `json:"field_type"`
	Description   string `json:"description"`
	EditableField bool   `json:"editable_field"`
	Optional      bool   `json:"optional"`
	Secret        bool   `json:"secret"`
}

// CatalogParamView is the flat table row of a catalog parameter.
type CatalogParamView struct {
	ParamID     string `json:"param_id"`
	FieldType   string `json:"field_type"`
	Optional    bool   `json:"optional"`
	Secret      bool   `json:"secret"`
	Description string `json:"description"`
}

// Views flattens the parameters of a provider catalog.
func (c *ProviderCatalog) Views() []CatalogParamView {
	views := make([]CatalogParamView, 0, len(c.Params))

	for _, p := range c.Params {
		views = append(views, CatalogParamView{
			ParamID:     p.ParamID,
			FieldType:   p.FieldType,
			Optional:    p.Optional,
			Secret:      p.Secret,
			Description: p.Description,
		})
	}

	return views
}
