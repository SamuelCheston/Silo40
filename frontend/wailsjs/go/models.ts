export namespace gorm {

	export class DeletedAt {
	    // Go type: time
	    Time: any;
	    Valid: boolean;

	    static createFrom(source: any = {}) {
	        return new DeletedAt(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], null);
	        this.Valid = source["Valid"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace llm {

	export class DecisionLog {
	    id: number;
	    seq: number;
	    kind: string;
	    year: number;
	    month: number;
	    actor_kind?: string;
	    actor_id?: number;
	    actor_name?: string;
	    candidates_json?: string;
	    factors_json?: string;
	    scores_json?: string;
	    final_choice: number;
	    source?: string;
	    shadow_choice?: number;
	    config_level?: string;
	    config_source?: string;
	    level?: string;
	    model?: string;
	    prompt_hash?: string;
	    validation?: string;
	    latency_ms?: number;
	    prompt_tokens?: number;
	    completion_tokens?: number;
	    // Go type: time
	    created_at: any;

	    static createFrom(source: any = {}) {
	        return new DecisionLog(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.seq = source["seq"];
	        this.kind = source["kind"];
	        this.year = source["year"];
	        this.month = source["month"];
	        this.actor_kind = source["actor_kind"];
	        this.actor_id = source["actor_id"];
	        this.actor_name = source["actor_name"];
	        this.candidates_json = source["candidates_json"];
	        this.factors_json = source["factors_json"];
	        this.scores_json = source["scores_json"];
	        this.final_choice = source["final_choice"];
	        this.source = source["source"];
	        this.shadow_choice = source["shadow_choice"];
	        this.config_level = source["config_level"];
	        this.config_source = source["config_source"];
	        this.level = source["level"];
	        this.model = source["model"];
	        this.prompt_hash = source["prompt_hash"];
	        this.validation = source["validation"];
	        this.latency_ms = source["latency_ms"];
	        this.prompt_tokens = source["prompt_tokens"];
	        this.completion_tokens = source["completion_tokens"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace model {

	export class StoryEvent {
	    id: string;
	    category: string;
	    title: string;
	    description: string;
	    type: string;

	    static createFrom(source: any = {}) {
	        return new StoryEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.type = source["type"];
	    }
	}
	export class ActionResult {
	    executed: boolean;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ActionResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.executed = source["executed"];
	        this.message = source["message"];
	    }
	}
	export class RumorView {
	    silo_id: number;
	    name: string;
	    status: string;
	    last_confirmed: number;
	    signal_source: string;
	    rebellion_band?: string;
	    population_band?: string;

	    static createFrom(source: any = {}) {
	        return new RumorView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo_id = source["silo_id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.last_confirmed = source["last_confirmed"];
	        this.signal_source = source["signal_source"];
	        this.rebellion_band = source["rebellion_band"];
	        this.population_band = source["population_band"];
	    }
	}
	export class WorldSiloRef {
	    id: number;
	    name: string;
	    is_player: boolean;
	    is_station: boolean;

	    static createFrom(source: any = {}) {
	        return new WorldSiloRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.is_player = source["is_player"];
	        this.is_station = source["is_station"];
	    }
	}
	export class Silo1Operation {
	    id: number;
	    target_silo_id?: number;
	    action_id: string;
	    label: string;
	    status: string;
	    start_year: number;
	    start_month: number;
	    due_year: number;
	    due_month: number;
	    effect: string;
	    trace_suspicion: number;
	    resolution?: string;

	    static createFrom(source: any = {}) {
	        return new Silo1Operation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.target_silo_id = source["target_silo_id"];
	        this.action_id = source["action_id"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.start_year = source["start_year"];
	        this.start_month = source["start_month"];
	        this.due_year = source["due_year"];
	        this.due_month = source["due_month"];
	        this.effect = source["effect"];
	        this.trace_suspicion = source["trace_suspicion"];
	        this.resolution = source["resolution"];
	    }
	}
	export class WorldPacing {
	    resource_rate: number;
	    ideology_rate: number;
	    political_rate: number;
	    action_rate: number;
	    demography_rate: number;

	    static createFrom(source: any = {}) {
	        return new WorldPacing(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resource_rate = source["resource_rate"];
	        this.ideology_rate = source["ideology_rate"];
	        this.political_rate = source["political_rate"];
	        this.action_rate = source["action_rate"];
	        this.demography_rate = source["demography_rate"];
	    }
	}
	export class WorldView {
	    id: number;
	    seed: number;
	    current_year: number;
	    current_month: number;
	    player_silo_id: number;
	    progress_stage: string;
	    pacing: WorldPacing;
	    silo1_alive: boolean;
	    silo1_simulated: boolean;
	    silo1_departments?: string[];
	    silo1_last_report?: string;
	    silo1_exposure: number;
	    silo1_counterintel_stage: number;
	    silo1_operations?: Silo1Operation[];
	    station_silo_id?: number;
	    station_silo?: Silo;
	    station_last_report?: string;
	    station_exposure: number;
	    station_counterintel_stage: number;
	    station_operations?: Silo1Operation[];
	    silos: WorldSiloRef[];
	    neighbors?: RumorView[];

	    static createFrom(source: any = {}) {
	        return new WorldView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.seed = source["seed"];
	        this.current_year = source["current_year"];
	        this.current_month = source["current_month"];
	        this.player_silo_id = source["player_silo_id"];
	        this.progress_stage = source["progress_stage"];
	        this.pacing = this.convertValues(source["pacing"], WorldPacing);
	        this.silo1_alive = source["silo1_alive"];
	        this.silo1_simulated = source["silo1_simulated"];
	        this.silo1_departments = source["silo1_departments"];
	        this.silo1_last_report = source["silo1_last_report"];
	        this.silo1_exposure = source["silo1_exposure"];
	        this.silo1_counterintel_stage = source["silo1_counterintel_stage"];
	        this.silo1_operations = this.convertValues(source["silo1_operations"], Silo1Operation);
	        this.station_silo_id = source["station_silo_id"];
	        this.station_silo = this.convertValues(source["station_silo"], Silo);
	        this.station_last_report = source["station_last_report"];
	        this.station_exposure = source["station_exposure"];
	        this.station_counterintel_stage = source["station_counterintel_stage"];
	        this.station_operations = this.convertValues(source["station_operations"], Silo1Operation);
	        this.silos = this.convertValues(source["silos"], WorldSiloRef);
	        this.neighbors = this.convertValues(source["neighbors"], RumorView);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlayerActionMeta {
	    id: string;
	    source: string;
	    event_id?: string;
	    label: string;
	    description: string;
	    group: string;
	    action_type: string;
	    target_type: string;
	    ap_cost: number;
	    duration_months: number;
	    enabled: boolean;
	    visibility: string;
	    disabled_reason?: string;
	    unavailable_behavior?: string;

	    static createFrom(source: any = {}) {
	        return new PlayerActionMeta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.event_id = source["event_id"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.group = source["group"];
	        this.action_type = source["action_type"];
	        this.target_type = source["target_type"];
	        this.ap_cost = source["ap_cost"];
	        this.duration_months = source["duration_months"];
	        this.enabled = source["enabled"];
	        this.visibility = source["visibility"];
	        this.disabled_reason = source["disabled_reason"];
	        this.unavailable_behavior = source["unavailable_behavior"];
	    }
	}
	export class AgentStats {
	    avg_connection: number;
	    prestige_base: number;
	    profession_factor: number;
	    trait_factor: number;
	    structural_prestige: number;
	    political_prestige: number;
	    ap_base_recovery: number;
	    ap_prestige_bonus: number;
	    ap_total_recovery: number;
	    ap_max: number;
	    propaganda_level: number;
	    propaganda_multiplier: number;
	    rebellion_base_effect: number;
	    is_faction_leader: boolean;

	    static createFrom(source: any = {}) {
	        return new AgentStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avg_connection = source["avg_connection"];
	        this.prestige_base = source["prestige_base"];
	        this.profession_factor = source["profession_factor"];
	        this.trait_factor = source["trait_factor"];
	        this.structural_prestige = source["structural_prestige"];
	        this.political_prestige = source["political_prestige"];
	        this.ap_base_recovery = source["ap_base_recovery"];
	        this.ap_prestige_bonus = source["ap_prestige_bonus"];
	        this.ap_total_recovery = source["ap_total_recovery"];
	        this.ap_max = source["ap_max"];
	        this.propaganda_level = source["propaganda_level"];
	        this.propaganda_multiplier = source["propaganda_multiplier"];
	        this.rebellion_base_effect = source["rebellion_base_effect"];
	        this.is_faction_leader = source["is_faction_leader"];
	    }
	}
	export class Connection {
	    id: number;
	    agent_id: number;
	    profession_id: number;
	    value: number;

	    static createFrom(source: any = {}) {
	        return new Connection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.agent_id = source["agent_id"];
	        this.profession_id = source["profession_id"];
	        this.value = source["value"];
	    }
	}
	export class Agent {
	    id: number;
	    user_id: number;
	    silo_id: number;
	    station_silo_id?: number;
	    name: string;
	    profession: string;
	    traits: string[];
	    political_prestige: number;
	    political_points: number;
	    action_points: number;
	    propaganda_level: number;
	    suspicion_level: number;
	    connections: Connection[];
	    known_fragments: string[];
	    relics?: Relic[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Agent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.user_id = source["user_id"];
	        this.silo_id = source["silo_id"];
	        this.station_silo_id = source["station_silo_id"];
	        this.name = source["name"];
	        this.profession = source["profession"];
	        this.traits = source["traits"];
	        this.political_prestige = source["political_prestige"];
	        this.political_points = source["political_points"];
	        this.action_points = source["action_points"];
	        this.propaganda_level = source["propaganda_level"];
	        this.suspicion_level = source["suspicion_level"];
	        this.connections = this.convertValues(source["connections"], Connection);
	        this.known_fragments = source["known_fragments"];
	        this.relics = this.convertValues(source["relics"], Relic);
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Faction {
	    id: number;
	    silo_id: number;
	    name: string;
	    signature: string;
	    tags: string[];
	    tag_stats?: Record<string, number>;
	    member_count: number;
	    representative_resident_id: number;
	    representative_cohort_id?: number;
	    representative_name: string;
	    influence: number;
	    cohesion: number;
	    prestige: number;
	    is_public: boolean;
	    stance?: string;
	    memory?: string[];
	    demands?: string[];
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Faction(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.name = source["name"];
	        this.signature = source["signature"];
	        this.tags = source["tags"];
	        this.tag_stats = source["tag_stats"];
	        this.member_count = source["member_count"];
	        this.representative_resident_id = source["representative_resident_id"];
	        this.representative_cohort_id = source["representative_cohort_id"];
	        this.representative_name = source["representative_name"];
	        this.influence = source["influence"];
	        this.cohesion = source["cohesion"];
	        this.prestige = source["prestige"];
	        this.is_public = source["is_public"];
	        this.stance = source["stance"];
	        this.memory = source["memory"];
	        this.demands = source["demands"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Resident {
	    id: number;
	    silo_id: number;
	    name: string;
	    cohort_id?: number;
	    profession_id: number;
	    profession: string;
	    faction_id?: number;
	    home_floor: number;
	    tags: string[];
	    loyalty: number;
	    ambition: number;
	    ideologies: Record<string, number>;
	    influence: number;
	    action_points: number;
	    suspicion_level: number;
	    political_prestige: number;
	    propaganda_level: number;
	    known_fragments: string[];
	    relations?: Record<string, number>;
	    is_representative: boolean;
	    alive: boolean;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Resident(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.name = source["name"];
	        this.cohort_id = source["cohort_id"];
	        this.profession_id = source["profession_id"];
	        this.profession = source["profession"];
	        this.faction_id = source["faction_id"];
	        this.home_floor = source["home_floor"];
	        this.tags = source["tags"];
	        this.loyalty = source["loyalty"];
	        this.ambition = source["ambition"];
	        this.ideologies = source["ideologies"];
	        this.influence = source["influence"];
	        this.action_points = source["action_points"];
	        this.suspicion_level = source["suspicion_level"];
	        this.political_prestige = source["political_prestige"];
	        this.propaganda_level = source["propaganda_level"];
	        this.known_fragments = source["known_fragments"];
	        this.relations = source["relations"];
	        this.is_representative = source["is_representative"];
	        this.alive = source["alive"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PopulationCohort {
	    id: number;
	    silo_id: number;
	    profession_id: number;
	    faction_id?: number;
	    name: string;
	    count: number;
	    ideology_profile: string[];
	    home_zone: string;
	    loyalty: number;
	    influence: number;
	    action_points: number;
	    political_prestige: number;
	    panic_sensitivity: number;
	    ideologies: Record<string, number>;
	    known_fragments?: string[];
	    tags?: string[];
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new PopulationCohort(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.profession_id = source["profession_id"];
	        this.faction_id = source["faction_id"];
	        this.name = source["name"];
	        this.count = source["count"];
	        this.ideology_profile = source["ideology_profile"];
	        this.home_zone = source["home_zone"];
	        this.loyalty = source["loyalty"];
	        this.influence = source["influence"];
	        this.action_points = source["action_points"];
	        this.political_prestige = source["political_prestige"];
	        this.panic_sensitivity = source["panic_sensitivity"];
	        this.ideologies = source["ideologies"];
	        this.known_fragments = source["known_fragments"];
	        this.tags = source["tags"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Floor {
	    id: number;
	    silo_id: number;
	    level: number;
	    function: string;
	    zone: string;
	    stability: number;
	    population: number;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Floor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.level = source["level"];
	        this.function = source["function"];
	        this.zone = source["zone"];
	        this.stability = source["stability"];
	        this.population = source["population"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Relic {
	    id: number;
	    silo_id: number;
	    name: string;
	    description: string;
	    source_dept: string;
	    discovery_year: number;
	    effects: Record<string, number>;
	    agent_id?: number;
	    profession_id?: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Relic(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.source_dept = source["source_dept"];
	        this.discovery_year = source["discovery_year"];
	        this.effects = source["effects"];
	        this.agent_id = source["agent_id"];
	        this.profession_id = source["profession_id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Profession {
	    id: number;
	    silo_id: number;
	    name: string;
	    class_type: string;
	    population: number;
	    ideologies: Record<string, number>;
	    panic_value: number;
	    productivity: number;
	    power_level: number;
	    zone: string;
	    known_fragments: string[];
	    relations?: Record<string, number>;
	    action_points: number;
	    suspicion_level: number;
	    political_prestige: number;
	    propaganda_level: number;
	    traits?: string[];
	    relics?: Relic[];
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Profession(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.name = source["name"];
	        this.class_type = source["class_type"];
	        this.population = source["population"];
	        this.ideologies = source["ideologies"];
	        this.panic_value = source["panic_value"];
	        this.productivity = source["productivity"];
	        this.power_level = source["power_level"];
	        this.zone = source["zone"];
	        this.known_fragments = source["known_fragments"];
	        this.relations = source["relations"];
	        this.action_points = source["action_points"];
	        this.suspicion_level = source["suspicion_level"];
	        this.political_prestige = source["political_prestige"];
	        this.propaganda_level = source["propaganda_level"];
	        this.traits = source["traits"];
	        this.relics = this.convertValues(source["relics"], Relic);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Resource {
	    id: number;
	    silo_id: number;
	    type: string;
	    amount: number;
	    net_balance: number;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Resource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.type = source["type"];
	        this.amount = source["amount"];
	        this.net_balance = source["net_balance"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EndingEntityContribution {
	    entity_id: string;
	    kind: string;
	    name: string;
	    scores: Record<string, number>;

	    static createFrom(source: any = {}) {
	        return new EndingEntityContribution(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entity_id = source["entity_id"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.scores = source["scores"];
	    }
	}
	export class EndingDecision {
	    config_version: number;
	    selected: string;
	    minimum_score: number;
	    scores: Record<string, number>;
	    contributions: EndingEntityContribution[];

	    static createFrom(source: any = {}) {
	        return new EndingDecision(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config_version = source["config_version"];
	        this.selected = source["selected"];
	        this.minimum_score = source["minimum_score"];
	        this.scores = source["scores"];
	        this.contributions = this.convertValues(source["contributions"], EndingEntityContribution);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistoryImpact {
	    id: number;
	    operation_id: number;
	    target_silo_id: number;
	    target_name: string;
	    action_id: string;
	    label: string;
	    year: number;
	    month: number;
	    legitimacy_delta: number;
	    cohesion_delta: number;
	    rebellion_delta: number;
	    pro_foreign_delta: number;
	    supplies_delta: number;
	    points: number;
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new HistoryImpact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.operation_id = source["operation_id"];
	        this.target_silo_id = source["target_silo_id"];
	        this.target_name = source["target_name"];
	        this.action_id = source["action_id"];
	        this.label = source["label"];
	        this.year = source["year"];
	        this.month = source["month"];
	        this.legitimacy_delta = source["legitimacy_delta"];
	        this.cohesion_delta = source["cohesion_delta"];
	        this.rebellion_delta = source["rebellion_delta"];
	        this.pro_foreign_delta = source["pro_foreign_delta"];
	        this.supplies_delta = source["supplies_delta"];
	        this.points = source["points"];
	        this.summary = source["summary"];
	    }
	}
	export class HistorySiloScore {
	    silo_id: number;
	    name: string;
	    points: number;
	    operation_count: number;
	    summary: string;
	    impacts?: HistoryImpact[];

	    static createFrom(source: any = {}) {
	        return new HistorySiloScore(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo_id = source["silo_id"];
	        this.name = source["name"];
	        this.points = source["points"];
	        this.operation_count = source["operation_count"];
	        this.summary = source["summary"];
	        this.impacts = this.convertValues(source["impacts"], HistoryImpact);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistoryInfluenceScore {
	    total: number;
	    grade: string;
	    summary: string;
	    silos: HistorySiloScore[];

	    static createFrom(source: any = {}) {
	        return new HistoryInfluenceScore(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.grade = source["grade"];
	        this.summary = source["summary"];
	        this.silos = this.convertValues(source["silos"], HistorySiloScore);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GameScore {
	    total: number;
	    survival_points: number;
	    diversity_points: number;
	    heritage_points: number;
	    ideology_points: number;
	    multiplier: number;
	    history_influence?: HistoryInfluenceScore;

	    static createFrom(source: any = {}) {
	        return new GameScore(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.survival_points = source["survival_points"];
	        this.diversity_points = source["diversity_points"];
	        this.heritage_points = source["heritage_points"];
	        this.ideology_points = source["ideology_points"];
	        this.multiplier = source["multiplier"];
	        this.history_influence = this.convertValues(source["history_influence"], HistoryInfluenceScore);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class VictoryStatus {
	    is_won: boolean;
	    type: string;
	    description: string;
	    score?: GameScore;
	    decision?: EndingDecision;

	    static createFrom(source: any = {}) {
	        return new VictoryStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_won = source["is_won"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.score = this.convertValues(source["score"], GameScore);
	        this.decision = this.convertValues(source["decision"], EndingDecision);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JudicialAttachment {
	    a_it: number;
	    a_mayor: number;
	    a_pact: number;
	    target_it: number;
	    target_mayor: number;
	    target_pact: number;
	    command_it: number;
	    command_mayor: number;
	    command_pact: number;
	    regime: string;
	    mayor_low_months: number;
	    mayor_high_months: number;
	    pact_low_months: number;
	    pact_qualified: boolean;
	    public_evidence_q: number;
	    independent_public_facts: number;
	    pact_locked: boolean;

	    static createFrom(source: any = {}) {
	        return new JudicialAttachment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.a_it = source["a_it"];
	        this.a_mayor = source["a_mayor"];
	        this.a_pact = source["a_pact"];
	        this.target_it = source["target_it"];
	        this.target_mayor = source["target_mayor"];
	        this.target_pact = source["target_pact"];
	        this.command_it = source["command_it"];
	        this.command_mayor = source["command_mayor"];
	        this.command_pact = source["command_pact"];
	        this.regime = source["regime"];
	        this.mayor_low_months = source["mayor_low_months"];
	        this.mayor_high_months = source["mayor_high_months"];
	        this.pact_low_months = source["pact_low_months"];
	        this.pact_qualified = source["pact_qualified"];
	        this.public_evidence_q = source["public_evidence_q"];
	        this.independent_public_facts = source["independent_public_facts"];
	        this.pact_locked = source["pact_locked"];
	    }
	}
	export class ShadowAuthority {
	    initialized: boolean;
	    initialization_source: string;
	    w: number;
	    k: number;
	    e: number;
	    d: number;
	    p: number;
	    x: number;
	    order_deviation: number;
	    police_obedience: number;
	    judicial: JudicialAttachment;

	    static createFrom(source: any = {}) {
	        return new ShadowAuthority(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.initialized = source["initialized"];
	        this.initialization_source = source["initialization_source"];
	        this.w = source["w"];
	        this.k = source["k"];
	        this.e = source["e"];
	        this.d = source["d"];
	        this.p = source["p"];
	        this.x = source["x"];
	        this.order_deviation = source["order_deviation"];
	        this.police_obedience = source["police_obedience"];
	        this.judicial = this.convertValues(source["judicial"], JudicialAttachment);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MayorOffice {
	    initialized: boolean;
	    holder_resident_id?: number;
	    holder_name: string;
	    legitimacy: number;

	    static createFrom(source: any = {}) {
	        return new MayorOffice(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.initialized = source["initialized"];
	        this.holder_resident_id = source["holder_resident_id"];
	        this.holder_name = source["holder_name"];
	        this.legitimacy = source["legitimacy"];
	    }
	}
	export class Silo {
	    id: number;
	    world_id: number;
	    name: string;
	    traits: string[];
	    safeguard_risk: number;
	    total_population: number;
	    legitimacy: number;
	    mayor: MayorOffice;
	    authority: ShadowAuthority;
	    cohesion: number;
	    rebellion: number;
	    dept_tension: number;
	    class_fragmentation: number;
	    history_burden: number;
	    event_trigger: number;
	    current_year: number;
	    current_month: number;
	    countdown: number;
	    silo1_destroyed: boolean;
	    victory_status?: VictoryStatus;
	    resources: Resource[];
	    professions: Profession[];
	    floors: Floor[];
	    cohorts?: PopulationCohort[];
	    residents?: Resident[];
	    factions?: Faction[];
	    relics?: Relic[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new Silo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.world_id = source["world_id"];
	        this.name = source["name"];
	        this.traits = source["traits"];
	        this.safeguard_risk = source["safeguard_risk"];
	        this.total_population = source["total_population"];
	        this.legitimacy = source["legitimacy"];
	        this.mayor = this.convertValues(source["mayor"], MayorOffice);
	        this.authority = this.convertValues(source["authority"], ShadowAuthority);
	        this.cohesion = source["cohesion"];
	        this.rebellion = source["rebellion"];
	        this.dept_tension = source["dept_tension"];
	        this.class_fragmentation = source["class_fragmentation"];
	        this.history_burden = source["history_burden"];
	        this.event_trigger = source["event_trigger"];
	        this.current_year = source["current_year"];
	        this.current_month = source["current_month"];
	        this.countdown = source["countdown"];
	        this.silo1_destroyed = source["silo1_destroyed"];
	        this.victory_status = this.convertValues(source["victory_status"], VictoryStatus);
	        this.resources = this.convertValues(source["resources"], Resource);
	        this.professions = this.convertValues(source["professions"], Profession);
	        this.floors = this.convertValues(source["floors"], Floor);
	        this.cohorts = this.convertValues(source["cohorts"], PopulationCohort);
	        this.residents = this.convertValues(source["residents"], Resident);
	        this.factions = this.convertValues(source["factions"], Faction);
	        this.relics = this.convertValues(source["relics"], Relic);
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ActionOutcome {
	    silo: Silo;
	    agent: Agent;
	    agent_stats: AgentStats;
	    available_actions?: PlayerActionMeta[];
	    world?: WorldView;
	    result: ActionResult;
	    logs: string[];
	    stories: StoryEvent[];
	    game_over: boolean;
	    ending_narrative?: string;

	    static createFrom(source: any = {}) {
	        return new ActionOutcome(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo = this.convertValues(source["silo"], Silo);
	        this.agent = this.convertValues(source["agent"], Agent);
	        this.agent_stats = this.convertValues(source["agent_stats"], AgentStats);
	        this.available_actions = this.convertValues(source["available_actions"], PlayerActionMeta);
	        this.world = this.convertValues(source["world"], WorldView);
	        this.result = this.convertValues(source["result"], ActionResult);
	        this.logs = source["logs"];
	        this.stories = this.convertValues(source["stories"], StoryEvent);
	        this.game_over = source["game_over"];
	        this.ending_narrative = source["ending_narrative"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	
	export class AgentAction {
	    type: string;
	    action_id?: string;
	    target_dept?: string;
	    fragment_ids?: string[];
	    profession_action?: string;
	    resource_target?: string;
	    cost: number;

	    static createFrom(source: any = {}) {
	        return new AgentAction(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.action_id = source["action_id"];
	        this.target_dept = source["target_dept"];
	        this.fragment_ids = source["fragment_ids"];
	        this.profession_action = source["profession_action"];
	        this.resource_target = source["resource_target"];
	        this.cost = source["cost"];
	    }
	}

	
	export class NeighborSiloConfig {
	    name: string;
	    agent_name: string;
	    profession: string;

	    static createFrom(source: any = {}) {
	        return new NeighborSiloConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.agent_name = source["agent_name"];
	        this.profession = source["profession"];
	    }
	}
	export class CreateGameRequest {
	    silo_name: string;
	    start_year: number;
	    trait_ids: string[];
	    agent_name: string;
	    profession: string;
	    seed?: number;
	    pacing?: WorldPacing;
	    neighbor?: NeighborSiloConfig;
	    include_silo1?: boolean;

	    static createFrom(source: any = {}) {
	        return new CreateGameRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo_name = source["silo_name"];
	        this.start_year = source["start_year"];
	        this.trait_ids = source["trait_ids"];
	        this.agent_name = source["agent_name"];
	        this.profession = source["profession"];
	        this.seed = source["seed"];
	        this.pacing = this.convertValues(source["pacing"], WorldPacing);
	        this.neighbor = this.convertValues(source["neighbor"], NeighborSiloConfig);
	        this.include_silo1 = source["include_silo1"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	
	export class StoryEventLog {
	    id: number;
	    silo_id: number;
	    timestamp: number;
	    year: number;
	    month: number;
	    source: string;
	    event_id: string;
	    category: string;
	    title: string;
	    description: string;
	    type: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new StoryEventLog(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.silo_id = source["silo_id"];
	        this.timestamp = source["timestamp"];
	        this.year = source["year"];
	        this.month = source["month"];
	        this.source = source["source"];
	        this.event_id = source["event_id"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EventHistoryResult {
	    events: StoryEventLog[];

	    static createFrom(source: any = {}) {
	        return new EventHistoryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], StoryEventLog);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EventQueueState {
	    silo: Silo;
	    agent: Agent;
	    agent_stats: AgentStats;
	    available_actions?: PlayerActionMeta[];
	    world?: WorldView;
	    game_over: boolean;
	    ending_narrative?: string;
	    pending_events: number;
	    pending_operation?: string;

	    static createFrom(source: any = {}) {
	        return new EventQueueState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo = this.convertValues(source["silo"], Silo);
	        this.agent = this.convertValues(source["agent"], Agent);
	        this.agent_stats = this.convertValues(source["agent_stats"], AgentStats);
	        this.available_actions = this.convertValues(source["available_actions"], PlayerActionMeta);
	        this.world = this.convertValues(source["world"], WorldView);
	        this.game_over = source["game_over"];
	        this.ending_narrative = source["ending_narrative"];
	        this.pending_events = source["pending_events"];
	        this.pending_operation = source["pending_operation"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EventStepResult {
	    silo: Silo;
	    agent: Agent;
	    agent_stats: AgentStats;
	    available_actions?: PlayerActionMeta[];
	    world?: WorldView;
	    game_over: boolean;
	    ending_narrative?: string;
	    pending_events: number;
	    pending_operation?: string;
	    processed_event_id?: string;
	    processed_event_type?: string;
	    operation_complete: boolean;
	    logs?: string[];
	    stories?: StoryEvent[];
	    action_result?: ActionResult;

	    static createFrom(source: any = {}) {
	        return new EventStepResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo = this.convertValues(source["silo"], Silo);
	        this.agent = this.convertValues(source["agent"], Agent);
	        this.agent_stats = this.convertValues(source["agent_stats"], AgentStats);
	        this.available_actions = this.convertValues(source["available_actions"], PlayerActionMeta);
	        this.world = this.convertValues(source["world"], WorldView);
	        this.game_over = source["game_over"];
	        this.ending_narrative = source["ending_narrative"];
	        this.pending_events = source["pending_events"];
	        this.pending_operation = source["pending_operation"];
	        this.processed_event_id = source["processed_event_id"];
	        this.processed_event_type = source["processed_event_type"];
	        this.operation_complete = source["operation_complete"];
	        this.logs = source["logs"];
	        this.stories = this.convertValues(source["stories"], StoryEvent);
	        this.action_result = this.convertValues(source["action_result"], ActionResult);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	

	export class ProfessionActionMeta {
	    id: string;
	    profession: string;
	    label: string;
	    description: string;
	    ap_cost: number;
	    target_type: string;
	    suspicion_penalty: number;

	    static createFrom(source: any = {}) {
	        return new ProfessionActionMeta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.profession = source["profession"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.ap_cost = source["ap_cost"];
	        this.target_type = source["target_type"];
	        this.suspicion_penalty = source["suspicion_penalty"];
	    }
	}
	export class GameState {
	    silo: Silo;
	    agent: Agent;
	    agent_stats: AgentStats;
	    game_over: boolean;
	    ending_narrative?: string;
	    victory_status?: VictoryStatus;
	    profession_actions?: ProfessionActionMeta[];
	    available_actions?: PlayerActionMeta[];
	    world?: WorldView;

	    static createFrom(source: any = {}) {
	        return new GameState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo = this.convertValues(source["silo"], Silo);
	        this.agent = this.convertValues(source["agent"], Agent);
	        this.agent_stats = this.convertValues(source["agent_stats"], AgentStats);
	        this.game_over = source["game_over"];
	        this.ending_narrative = source["ending_narrative"];
	        this.victory_status = this.convertValues(source["victory_status"], VictoryStatus);
	        this.profession_actions = this.convertValues(source["profession_actions"], ProfessionActionMeta);
	        this.available_actions = this.convertValues(source["available_actions"], PlayerActionMeta);
	        this.world = this.convertValues(source["world"], WorldView);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	

	

	

	

	

	

	

	

	

	export class TickResult {
	    silo: Silo;
	    agent: Agent;
	    agent_stats: AgentStats;
	    available_actions?: PlayerActionMeta[];
	    world?: WorldView;
	    logs: string[];
	    stories: StoryEvent[];
	    game_over: boolean;
	    ending_narrative?: string;

	    static createFrom(source: any = {}) {
	        return new TickResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.silo = this.convertValues(source["silo"], Silo);
	        this.agent = this.convertValues(source["agent"], Agent);
	        this.agent_stats = this.convertValues(source["agent_stats"], AgentStats);
	        this.available_actions = this.convertValues(source["available_actions"], PlayerActionMeta);
	        this.world = this.convertValues(source["world"], WorldView);
	        this.logs = source["logs"];
	        this.stories = this.convertValues(source["stories"], StoryEvent);
	        this.game_over = source["game_over"];
	        this.ending_narrative = source["ending_narrative"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	


}

export namespace service {

	export class LlmStatus {
	    enabled: boolean;
	    mode: string;
	    level: string;
	    model?: string;
	    log_entries: number;
	    advisor_error?: string;

	    static createFrom(source: any = {}) {
	        return new LlmStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.level = source["level"];
	        this.model = source["model"];
	        this.log_entries = source["log_entries"];
	        this.advisor_error = source["advisor_error"];
	    }
	}

}

