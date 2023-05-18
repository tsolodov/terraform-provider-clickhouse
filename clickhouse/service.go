package clickhouse

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &serviceResource{}
	_ resource.ResourceWithConfigure = &serviceResource{}
)

// NewServiceResource is a helper function to simplify the provider implementation.
func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

// serviceResource is the resource implementation.
type serviceResource struct {
	client *Client
}

type serviceResourceModel struct {
	ID                 types.String    `tfsdk:"id"`
	Name               types.String    `tfsdk:"name"`
	CloudProvider      types.String    `tfsdk:"cloud_provider"`
	Region             types.String    `tfsdk:"region"`
	Tier               types.String    `tfsdk:"tier"`
	IdleScaling        types.Bool      `tfsdk:"idle_scaling"`
	IpAccessList       []IpAccessModel `tfsdk:"ip_access"`
	MinTotalMemoryGb   types.Int64     `tfsdk:"min_total_memory_gb"`
	MaxTotalMemoryGb   types.Int64     `tfsdk:"max_total_memory_gb"`
	IdleTimeoutMinutes types.Int64     `tfsdk:"idle_timeout_minutes"`
	LastUpdated        types.String    `tfsdk:"last_updated"`
}

type IpAccessModel struct {
	Source      types.String `tfsdk:"source"`
	Description types.String `tfsdk:"description"`
}

// Metadata returns the resource type name.
func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines the schema for the resource.
func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"cloud_provider": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"tier": schema.StringAttribute{
				Required: true,
			},
			"idle_scaling": schema.BoolAttribute{
				Required: true,
			},
			"ip_access": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
			"min_total_memory_gb": schema.Int64Attribute{
				Required: true,
			},
			"max_total_memory_gb": schema.Int64Attribute{
				Required: true,
			},
			"idle_timeout_minutes": schema.Int64Attribute{
				Required: true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*Client)
}

// Create a new resource
func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan serviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	service := Service{
		Name:               string(plan.Name.ValueString()),
		Provider:           string(plan.CloudProvider.ValueString()),
		Region:             string(plan.Region.ValueString()),
		Tier:               string(plan.Tier.ValueString()),
		IdleScaling:        bool(plan.IdleScaling.ValueBool()),
		MinTotalMemoryGb:   int(plan.MinTotalMemoryGb.ValueInt64()),
		MaxTotalMemoryGb:   int(plan.MaxTotalMemoryGb.ValueInt64()),
		IdleTimeoutMinutes: int(plan.IdleTimeoutMinutes.ValueInt64()),
	}
	for _, item := range plan.IpAccessList {
		service.IpAccessList = append(service.IpAccessList, IpAccess{
			Source:      string(item.Source.ValueString()),
			Description: string(item.Description.ValueString()),
		})
	}

	// Create new service
	s, err := r.client.CreateService(service)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating service",
			"Could not create service, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(s.Id)
	plan.Name = types.StringValue(s.Name)
	plan.CloudProvider = types.StringValue(s.Provider)
	plan.Region = types.StringValue(s.Region)
	plan.Tier = types.StringValue(s.Tier)
	plan.IdleScaling = types.BoolValue(s.IdleScaling)
	plan.MinTotalMemoryGb = types.Int64Value(int64(s.MinTotalMemoryGb))
	plan.MaxTotalMemoryGb = types.Int64Value(int64(s.MaxTotalMemoryGb))
	plan.IdleTimeoutMinutes = types.Int64Value(int64(s.IdleTimeoutMinutes))
	for ipAccessIndex, ipAccess := range s.IpAccessList {
		plan.IpAccessList[ipAccessIndex] = IpAccessModel{
			Source:      types.StringValue(ipAccess.Source),
			Description: types.StringValue(ipAccess.Description),
		}
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed service value from HashiCups
	service, err := r.client.GetService(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Service",
			"Could not read ClickHouse service id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.IpAccessList = []IpAccessModel{}
	for _, item := range service.IpAccessList {
		state.IpAccessList = append(state.IpAccessList, IpAccessModel{
			Source:      types.StringValue(item.Source),
			Description: types.StringValue(item.Description),
		})
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan, state serviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if plan.CloudProvider != state.CloudProvider {
		resp.Diagnostics.AddAttributeError(
			path.Root("cloud_provider"),
			"Invalid Update",
			"ClickHouse does not support changing service cloud providers",
		)
	}

	if plan.Region != state.Region {
		resp.Diagnostics.AddAttributeError(
			path.Root("region"),
			"Invalid Update",
			"ClickHouse does not support changing service regions",
		)
	}

	if plan.Tier != state.Tier {
		resp.Diagnostics.AddAttributeError(
			path.Root("tier"),
			"Invalid Update",
			"ClickHouse does not support changing service tiers",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	serviceId := state.ID.ValueString()
	service := ServiceUpdate{
		Name:         "",
		IpAccessList: nil,
	}
	serviceChange := false

	if plan.Name != state.Name {
		service.Name = plan.Name.ValueString()
		serviceChange = true
	}
	if !equal(plan.IpAccessList, state.IpAccessList) {
		serviceChange = true
		ipAccessListRawOld := state.IpAccessList
		ipAccessListRawNew := plan.IpAccessList

		ipAccessListOld := []IpAccess{}
		ipAccessListNew := []IpAccess{}

		for _, item := range ipAccessListRawOld {
			ipAccess := IpAccess{
				Source:      item.Source.ValueString(),
				Description: item.Description.ValueString(),
			}

			ipAccessListOld = append(ipAccessListOld, ipAccess)
		}

		for _, item := range ipAccessListRawNew {
			ipAccess := IpAccess{
				Source:      item.Source.ValueString(),
				Description: item.Description.ValueString(),
			}

			ipAccessListNew = append(ipAccessListNew, ipAccess)
		}

		add, remove := diffArrays(ipAccessListOld, ipAccessListNew, func(a IpAccess) string {
			return a.Source
		})

		service.IpAccessList = &IpAccessUpdate{
			Add:    add,
			Remove: remove,
		}
	}

	// Update existing order
	var s *Service
	if serviceChange {
		var err error
		s, err = r.client.UpdateService(serviceId, service)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating ClickHouse Service",
				"Could not update service, unexpected error: "+err.Error(),
			)
			return
		}
	}

	scalingChange := false
	serviceScaling := ServiceScalingUpdate{}

	if plan.IdleScaling != state.IdleScaling {
		scalingChange = true
		idleScaling := new(bool)
		*idleScaling = plan.IdleScaling.ValueBool()
		serviceScaling.IdleScaling = idleScaling
	}
	if plan.MinTotalMemoryGb != state.MinTotalMemoryGb {
		scalingChange = true
		serviceScaling.MinTotalMemoryGb = int(plan.MinTotalMemoryGb.ValueInt64())
	}
	if plan.MaxTotalMemoryGb != state.MaxTotalMemoryGb {
		scalingChange = true
		serviceScaling.MaxTotalMemoryGb = int(plan.MaxTotalMemoryGb.ValueInt64())
	}
	if plan.IdleTimeoutMinutes != state.IdleTimeoutMinutes {
		scalingChange = true
		serviceScaling.IdleTimeoutMinutes = int(plan.IdleTimeoutMinutes.ValueInt64())
	}

	if scalingChange {
		var err error
		s, err = r.client.UpdateServiceScaling(serviceId, serviceScaling)
		if err != nil {
			// resetValue(d, "idle_scaling")
			// resetValue(d, "min_total_memory_gb")
			// resetValue(d, "max_total_memory_gb")
			// resetValue(d, "idle_timeout_minutes")
			resp.Diagnostics.AddError(
				"Error Updating ClickHouse Service Scaling",
				"Could not update service scaling, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Update resource state with updated items and timestamp
	plan.ID = types.StringValue(s.Id)
	plan.Name = types.StringValue(s.Name)
	plan.CloudProvider = types.StringValue(s.Provider)
	plan.Region = types.StringValue(s.Region)
	plan.Tier = types.StringValue(s.Tier)
	plan.IdleScaling = types.BoolValue(s.IdleScaling)
	plan.MinTotalMemoryGb = types.Int64Value(int64(s.MinTotalMemoryGb))
	plan.MaxTotalMemoryGb = types.Int64Value(int64(s.MaxTotalMemoryGb))
	plan.IdleTimeoutMinutes = types.Int64Value(int64(s.IdleTimeoutMinutes))
	for ipAccessIndex, ipAccess := range s.IpAccessList {
		plan.IpAccessList[ipAccessIndex] = IpAccessModel{
			Source:      types.StringValue(ipAccess.Source),
			Description: types.StringValue(ipAccess.Description),
		}
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state serviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.DeleteService(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Service",
			"Could not delete service, unexpected error: "+err.Error(),
		)
		return
	}
}
