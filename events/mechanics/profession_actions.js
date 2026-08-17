defineMechanics([
  professionAction({
    id: "MAYOR_PUBLIC_ADDRESS",
    profession: "Mayor",
    label: "Public Address",
    description: "Deliver a rousing speech to the silo. Legitimacy +6%, all departments panic -4%, cohesion +3%.",
    ap_cost: 15,
    target_type: "NONE",
    suspicion_penalty: 0.01,
    apply(ctx) {
      return {
        mutations: [
          { type: "cohort_ideology_delta_all", ideology: "loyalty", value: 0.06 },
          ...allProfessionMutations(ctx, "panic_value", -0.04),
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.01 },
        ],
        action_result: { executed: true, message: "Public address delivered. Population loyalty increased and panic eased." },
      };
    },
  }),
  professionAction({
    id: "MAYOR_DIRECT_ORDER",
    profession: "Mayor",
    label: "Direct Order",
    description: "Issue an executive order to a department. Target productivity +10%, panic -5%.",
    ap_cost: 20,
    target_type: "DEPT",
    suspicion_penalty: 0.02,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "profession_metric_delta", profession: target.name, field: "productivity", value: 0.10 },
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: -0.05 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.02 },
        ],
        action_result: { executed: true, message: "Executive order issued. " + target.name + " productivity improved." },
      };
    },
  }),
  professionAction({
    id: "JUDICIAL_SEARCH_WARRANT",
    profession: "Judicial",
    label: "Search Warrant",
    description: "Serve a search warrant on a department to seize intel. Gain 1 fragment; target panic +3%.",
    ap_cost: 15,
    target_type: "DEPT",
    suspicion_penalty: 0.02,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      const unknown = unknownFragmentsFrom(ctx, target.name);
      if (unknown.length === 0) {
        return { action_result: { executed: false, message: "Nothing new was found while searching " + target.name + "." } };
      }
      const fragment = unknown[randomInt(unknown.length)];
      return {
        mutations: [
          { type: "actor_fragment_add", fragment },
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: 0.03 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.02 },
        ],
        action_result: { executed: true, message: "Search warrant executed. Seized intel on " + fragment + " from " + target.name + "." },
      };
    },
  }),
  professionAction({
    id: "JUDICIAL_ARREST",
    profession: "Judicial",
    label: "Arrest",
    description: "Arrest key figures in a department. Target action points -20, panic +10%, legitimacy +4%.",
    ap_cost: 25,
    target_type: "DEPT",
    suspicion_penalty: 0.03,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "profession_metric_delta", profession: target.name, field: "action_points", value: -20 },
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: 0.10 },
          { type: "cohort_ideology_delta_all", ideology: "loyalty", value: 0.04 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.03 },
        ],
        action_result: { executed: true, message: "Arrests carried out in " + target.name + ". The law is reaffirmed." },
      };
    },
  }),
  professionAction({
    id: "IT_SURVEILLANCE",
    profession: "IT",
    label: "Surveillance",
    description: "Place a department under full surveillance. Connection +15% with the target (leverage of fear).",
    ap_cost: 15,
    target_type: "DEPT",
    suspicion_penalty: 0,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "actor_connection_delta", connection_dept: target.name, value: 0.15 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
        ],
        action_result: { executed: true, message: target.name + " is now under surveillance. You hold leverage over them." },
      };
    },
  }),
  professionAction({
    id: "IT_COVER_UP",
    profession: "IT",
    label: "Cover-Up",
    description: "Erase sensitive records from a department. Remove 1-2 fragments; safeguard risk +3%.",
    ap_cost: 25,
    target_type: "DEPT",
    suspicion_penalty: 0,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      const known = (target.known_fragments || []).slice();
      if (known.length === 0) {
        return { action_result: { executed: false, message: target.name + " holds no records worth erasing." } };
      }
      const removeCount = randomFloat() < 0.5 ? 2 : 1;
      const fragments = [];
      const pool = known.slice();
      for (let i = 0; i < removeCount && pool.length > 0; i++) {
        const index = randomInt(pool.length);
        fragments.push(pool[index]);
        pool.splice(index, 1);
      }
      return {
        mutations: [
          ...fragments.map((fragment) => ({ type: "profession_fragment_remove", profession: target.name, fragment })),
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: 0.05 },
          { type: "silo_metric_delta", metric: "safeguard_risk", value: 0.03 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
        ],
        action_result: { executed: true, message: "Records erased from " + target.name + ": " + fragments.join(", ") + ". Safeguard risk grows..." },
      };
    },
  }),
  professionAction({
    id: "POLICE_INTERROGATE",
    profession: "Police",
    label: "Interrogate",
    description: "Interrogate detainees from a department. Gain 1 fragment; target panic +5%.",
    ap_cost: 15,
    target_type: "DEPT",
    suspicion_penalty: 0.02,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      const unknown = unknownFragmentsFrom(ctx, target.name);
      if (unknown.length === 0) {
        return { action_result: { executed: false, message: "Interrogation yielded nothing new from " + target.name + "." } };
      }
      const fragment = unknown[randomInt(unknown.length)];
      return {
        mutations: [
          { type: "actor_fragment_add", fragment },
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: 0.05 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.02 },
        ],
        action_result: { executed: true, message: "Interrogation extracted intel on " + fragment + " from " + target.name + "." },
      };
    },
  }),
  professionAction({
    id: "POLICE_CRACKDOWN",
    profession: "Police",
    label: "Crackdown",
    description: "Suppress a department by force. Target panic -15%, ideology -5%, productivity -3%.",
    ap_cost: 25,
    target_type: "DEPT",
    suspicion_penalty: 0.03,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: -0.15 },
          { type: "profession_metric_delta", profession: target.name, field: "productivity", value: -0.03 },
          { type: "cohort_ideology_delta_all", profession: target.name, ideology: "pro_foreign", value: -0.05 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.03 },
        ],
        action_result: { executed: true, message: "Crackdown executed in " + target.name + ". Order restored, at a cost." },
      };
    },
  }),
  professionAction({
    id: "MEDICAL_TREAT",
    profession: "Medical",
    label: "Community Treatment",
    description: "Deploy medics to a department. Target panic -12%, productivity +5%.",
    ap_cost: 15,
    target_type: "DEPT",
    suspicion_penalty: 0.01,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: -0.12 },
          { type: "profession_metric_delta", profession: target.name, field: "productivity", value: 0.05 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.01 },
        ],
        action_result: { executed: true, message: "Medics deployed to " + target.name + ". Panic eased and health improved." },
      };
    },
  }),
  professionAction({
    id: "MEDICAL_QUARANTINE",
    profession: "Medical",
    label: "Quarantine",
    description: "Quarantine a department for its own safety. Target panic -20%, productivity -12% and ideology -4%.",
    ap_cost: 20,
    target_type: "DEPT",
    suspicion_penalty: 0.02,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: -0.20 },
          { type: "profession_metric_delta", profession: target.name, field: "productivity", value: -0.12 },
          { type: "cohort_ideology_delta_all", profession: target.name, ideology: "pro_foreign", value: -0.04 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.02 },
        ],
        action_result: { executed: true, message: target.name + " placed under quarantine. The silence is oppressive." },
      };
    },
  }),
  professionAction({
    id: "SUPPLY_RATION",
    profession: "Supply",
    label: "Ration Allocation",
    description: "Reallocate stockpiles. Add +1000 to a chosen resource.",
    ap_cost: 15,
    target_type: "RESOURCE",
    suspicion_penalty: 0.01,
    apply(ctx) {
      const resource = ctx.action.resource_target;
      if (!resource) {
        return { action_result: { executed: false, message: "Invalid resource target." } };
      }
      return {
        mutations: [
          { type: "resource_delta", resource, value: 1000 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.01 },
        ],
        action_result: { executed: true, message: "Reallocated stockpiles. +1000 " + resource + "." },
      };
    },
  }),
  professionAction({
    id: "SUPPLY_SHELTER",
    profession: "Supply",
    label: "Shelter",
    description: "Smuggle a department under your protection. Target panic -10%, connection +15%, productivity +5%.",
    ap_cost: 20,
    target_type: "DEPT",
    suspicion_penalty: 0.01,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      return {
        mutations: [
          { type: "profession_metric_delta", profession: target.name, field: "panic_value", value: -0.10 },
          { type: "profession_metric_delta", profession: target.name, field: "productivity", value: 0.05 },
          { type: "actor_connection_delta", connection_dept: target.name, value: 0.15 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.01 },
        ],
        action_result: { executed: true, message: target.name + " is now sheltered by the Supply network." },
      };
    },
  }),
  professionAction({
    id: "MECHANICAL_OVERHAUL",
    profession: "Mechanical",
    label: "Overhaul",
    description: "Overhaul the silo machinery. Energy +500, Materials +200, own productivity +5%.",
    ap_cost: 15,
    target_type: "NONE",
    suspicion_penalty: 0.01,
    apply(ctx) {
      return {
        mutations: [
          { type: "resource_delta", resource: "Energy", value: 500 },
          { type: "resource_delta", resource: "Materials", value: 200 },
          { type: "profession_metric_delta", profession: ctx.actor.profession, field: "productivity", value: 0.05 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.01 },
        ],
        action_result: { executed: true, message: "Machinery overhauled. Energy and materials production improved." },
      };
    },
  }),
  professionAction({
    id: "MECHANICAL_PIPE_TAP",
    profession: "Mechanical",
    label: "Pipe Tap",
    description: "Eavesdrop through the pipes that carry every whisper. Gain 1 fragment from a department.",
    ap_cost: 15,
    target_type: "DEPT",
    suspicion_penalty: 0.02,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      const unknown = unknownFragmentsFrom(ctx, target.name);
      if (unknown.length === 0) {
        return { action_result: { executed: false, message: "The pipes carried nothing new about " + target.name + "." } };
      }
      const fragment = unknown[randomInt(unknown.length)];
      return {
        mutations: [
          { type: "actor_fragment_add", fragment },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.02 },
        ],
        action_result: { executed: true, message: "Eavesdropped through the pipes. Learned about " + fragment + " from " + target.name + "." },
      };
    },
  }),
  professionAction({
    id: "MINES_DEEP_EXCAVATION",
    profession: "Mines",
    label: "Deep Excavation",
    description: "Push the mines deeper. Materials +800, own productivity +5%.",
    ap_cost: 15,
    target_type: "NONE",
    suspicion_penalty: 0.005,
    apply(ctx) {
      return {
        mutations: [
          { type: "resource_delta", resource: "Materials", value: 800 },
          { type: "profession_metric_delta", profession: ctx.actor.profession, field: "productivity", value: 0.05 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.005 },
        ],
        action_result: { executed: true, message: "Deep excavation completed. Materials reserves increased." },
      };
    },
  }),
  professionAction({
    id: "MINES_TUNNEL_NETWORK",
    profession: "Mines",
    label: "Tunnel Network",
    description: "Spin a network through the lower tunnels. All commoner departments connection +10%, ideology +3%.",
    ap_cost: 20,
    target_type: "NONE",
    suspicion_penalty: 0.005,
    apply(ctx) {
      const mutations = [
        { type: "cohort_ideology_delta_all", class_type: "COMMONER", ideology: "pro_foreign", value: 0.03 },
        { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
        { type: "actor_metric_delta", field: "suspicion_level", value: 0.005 },
      ];
      for (const profession of ctx.silo.professions || []) {
        if (profession.class_type === "COMMONER") {
          mutations.push({ type: "actor_connection_delta", connection_dept: profession.name, value: 0.10 });
        }
      }
      return {
        mutations,
        action_result: { executed: true, message: "The tunnel network hums with new alliances and whispered hopes." },
      };
    },
  }),
  professionAction({
    id: "AGRICULTURAL_INTENSIVE_HARVEST",
    profession: "Agricultural",
    label: "Intensive Harvest",
    description: "Work the hydroponics around the clock. Supplies +1500, own productivity +8%.",
    ap_cost: 15,
    target_type: "NONE",
    suspicion_penalty: 0.01,
    apply(ctx) {
      return {
        mutations: [
          { type: "resource_delta", resource: "Supplies", value: 1500 },
          { type: "profession_metric_delta", profession: ctx.actor.profession, field: "productivity", value: 0.08 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.01 },
        ],
        action_result: { executed: true, message: "Intensive harvest completed. Supplies increased." },
      };
    },
  }),
  professionAction({
    id: "AGRICULTURAL_FIELD_GOSSIP",
    profession: "Agricultural",
    label: "Field Gossip",
    description: "Let rumors ripen in the fields. Gain 1 fragment from a department; target ideology +3%.",
    ap_cost: 10,
    target_type: "DEPT",
    suspicion_penalty: 0.015,
    apply(ctx) {
      const target = targetProfession(ctx);
      if (!target) return missingTarget();
      const unknown = unknownFragmentsFrom(ctx, target.name);
      if (unknown.length === 0) {
        return { action_result: { executed: false, message: "The fields whispered nothing new about " + target.name + "." } };
      }
      const fragment = unknown[randomInt(unknown.length)];
      return {
        mutations: [
          { type: "actor_fragment_add", fragment },
          { type: "cohort_ideology_delta_all", profession: target.name, ideology: "pro_foreign", value: 0.03 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          { type: "actor_metric_delta", field: "suspicion_level", value: 0.015 },
        ],
        action_result: { executed: true, message: "Rumors spread from the fields. Heard about " + fragment + " from " + target.name + "." },
      };
    },
  }),
]);

function professionAction(def) {
  return {
    id: def.id,
    profession_action: def.id,
    profession: def.profession,
    label: def.label,
    description: def.description,
    ap_cost: def.ap_cost,
    target_type: def.target_type,
    duration_months: 1,
    suspicion_penalty: def.suspicion_penalty,
    apply: def.apply,
  };
}

function targetProfession(ctx) {
  return professionByName(ctx, ctx.action.target_dept || "");
}

function missingTarget() {
  return { action_result: { executed: false, message: "Target department not found." } };
}

function allProfessionMutations(ctx, field, value) {
  return (ctx.silo.professions || []).map((profession) => ({
    type: "profession_metric_delta",
    profession: profession.name,
    field,
    value,
  }));
}
