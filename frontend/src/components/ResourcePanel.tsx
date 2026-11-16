import { memo } from "react";
import { HStack, Text, VStack } from "@chakra-ui/react";
import { Silo, Resource } from "../logic/models";
import { BatteryCharging, Boxes, Wheat } from "lucide-react";
import { Tooltip } from "./ui/tooltip";

// 三种资源固定展示顺序：缺什么一眼看得出，不随后端数组顺序漂移。
const RESOURCE_ORDER = ["Energy", "Materials", "Supplies"];

const RESOURCE_META: Record<string, { label: string; icon: typeof BatteryCharging; note: string }> = {
    Energy: { label: "能源", icon: BatteryCharging, note: "机械部门产出，供居民生活与农业转化为补给。" },
    Materials: { label: "材料", icon: Boxes, note: "矿业部门产出，供居民生活与供应部门转化为补给。" },
    Supplies: { label: "补给", icon: Wheat, note: "农业与补给部门产出，养活全体人口。" },
};

// 库存与结余同屏：库存归零会直接抬升死亡率，结余为负说明库存正在被消耗。
// 后端 net_balance 是年度速率（资源增量 = 结余 × 年数），界面按月呈现。
function resourceState(res: Resource): { color: string; hint: string } {
    if (res.amount <= 0) {
        return { color: "red.600", hint: "库存已耗尽：死亡率正在上升。" };
    }
    if (res.net_balance < 0) {
        return { color: "orange.600", hint: "入不敷出：库存正在被消耗。" };
    }
    return { color: "green.600", hint: "本月结算有盈余或持平。" };
}

interface ResourcePanelProps {
    silo: Silo | null;
}

export const ResourcePanel = memo(function ResourcePanel({ silo }: ResourcePanelProps) {
    if (!silo || !silo.resources || silo.resources.length === 0) return null;

    const byType: Record<string, Resource> = {};
    for (const res of silo.resources) {
        byType[res.type] = res;
    }
    const ordered = RESOURCE_ORDER
        .map(type => byType[type])
        .filter((res): res is Resource => Boolean(res));

    return (
        <HStack gap={6} wrap="wrap">
            {ordered.map(res => {
                const meta = RESOURCE_META[res.type] || { label: res.type, icon: Boxes, note: "" };
                const Icon = meta.icon;
                const { color, hint } = resourceState(res);
                const monthlyNet = (res.net_balance || 0) / 12;
                const netText = `${monthlyNet >= 0 ? "+" : ""}${monthlyNet.toFixed(1)}`;
                return (
                    <Tooltip
                        key={res.type}
                        content={
                            <VStack align="start" gap={1} p={1}>
                                <Text fontWeight="bold" borderBottom="1px solid" borderColor="whiteAlpha.300" w="full" pb={1}>
                                    {meta.label}
                                </Text>
                                <HStack justify="space-between" w="full">
                                    <Text>库存:</Text>
                                    <Text>{Math.round(res.amount)}</Text>
                                </HStack>
                                <HStack justify="space-between" w="full">
                                    <Text>月度结余:</Text>
                                    <Text color={monthlyNet < 0 ? "red.300" : "green.300"}>{netText}</Text>
                                </HStack>
                                <Text fontSize="10px" color="gray.400" mt={1}>{meta.note}</Text>
                                <Text fontSize="10px" color={res.amount <= 0 ? "red.300" : "gray.400"}>{hint}</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Icon size={16} color={color === "red.600" ? "#e53e3e" : color === "orange.600" ? "#dd6b20" : "#38a169"} />
                            <Text fontSize="xs" color="gray.500">{meta.label}</Text>
                            <Text fontSize="sm" fontWeight="bold" color={color}>
                                {Math.round(res.amount)}
                            </Text>
                            <Text fontSize="xs" color={monthlyNet < 0 ? "orange.600" : "gray.500"}>
                                {netText}/月
                            </Text>
                        </HStack>
                    </Tooltip>
                );
            })}
        </HStack>
    );
});
