package terraform

import (
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
)

func regionToTerraform(r monitorv1.Region) string {
	switch r {
	// Fly.io (18)
	case monitorv1.Region_REGION_FLY_AMS:
		return "fly-ams"
	case monitorv1.Region_REGION_FLY_ARN:
		return "fly-arn"
	case monitorv1.Region_REGION_FLY_BOM:
		return "fly-bom"
	case monitorv1.Region_REGION_FLY_CDG:
		return "fly-cdg"
	case monitorv1.Region_REGION_FLY_DFW:
		return "fly-dfw"
	case monitorv1.Region_REGION_FLY_EWR:
		return "fly-ewr"
	case monitorv1.Region_REGION_FLY_FRA:
		return "fly-fra"
	case monitorv1.Region_REGION_FLY_GRU:
		return "fly-gru"
	case monitorv1.Region_REGION_FLY_IAD:
		return "fly-iad"
	case monitorv1.Region_REGION_FLY_JNB:
		return "fly-jnb"
	case monitorv1.Region_REGION_FLY_LAX:
		return "fly-lax"
	case monitorv1.Region_REGION_FLY_LHR:
		return "fly-lhr"
	case monitorv1.Region_REGION_FLY_NRT:
		return "fly-nrt"
	case monitorv1.Region_REGION_FLY_ORD:
		return "fly-ord"
	case monitorv1.Region_REGION_FLY_SJC:
		return "fly-sjc"
	case monitorv1.Region_REGION_FLY_SIN:
		return "fly-sin"
	case monitorv1.Region_REGION_FLY_SYD:
		return "fly-syd"
	case monitorv1.Region_REGION_FLY_YYZ:
		return "fly-yyz"
	// Koyeb (6)
	case monitorv1.Region_REGION_KOYEB_FRA:
		return "koyeb-fra"
	case monitorv1.Region_REGION_KOYEB_PAR:
		return "koyeb-par"
	case monitorv1.Region_REGION_KOYEB_SFO:
		return "koyeb-sfo"
	case monitorv1.Region_REGION_KOYEB_SIN:
		return "koyeb-sin"
	case monitorv1.Region_REGION_KOYEB_TYO:
		return "koyeb-tyo"
	case monitorv1.Region_REGION_KOYEB_WAS:
		return "koyeb-was"
	// Railway (4)
	case monitorv1.Region_REGION_RAILWAY_US_WEST2:
		return "railway-us-west2"
	case monitorv1.Region_REGION_RAILWAY_US_EAST4:
		return "railway-us-east4"
	case monitorv1.Region_REGION_RAILWAY_EUROPE_WEST4:
		return "railway-europe-west4"
	case monitorv1.Region_REGION_RAILWAY_ASIA_SOUTHEAST1:
		return "railway-asia-southeast1"
	default:
		return r.String()
	}
}
