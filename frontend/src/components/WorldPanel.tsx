import React from 'react';
import { Badge, Box, Heading, HStack, Text, VStack } from '@chakra-ui/react';

interface WorldPanelProps {
    world: any;
    agent: any;
}

const statusText: Record<string, string> = {
    normal: '运转中',
    unrest: '动荡',
    silent: '失联',
    destroyed: '覆灭',
};

const operationStatusText: Record<string, string> = {
    PENDING: '在途',
    RESOLVED: '已生效',
    SUPERSEDED: '未再生效',
    FAILED: '失败',
};

const nextCounterintelThreshold = (exposure: number) => exposure >= 0.75 ? 'MAX' : `${(Math.floor(exposure * 4) + 1) * 25}%`;

export const WorldPanel: React.FC<WorldPanelProps> = ({ world, agent }) => {
    if (!world) return null;
    const suspicion = agent?.suspicion_level || 0;
    const counterintelActive = (world.silo1_counterintel_stage || 0) > 0 || (world.station_counterintel_stage || 0) > 0;

    return (
        <Box w="full" p={5} bg="blue.50" borderRadius="md" border="1px solid" borderColor="blue.200">
            <HStack justify="space-between" align="start" mb={3}>
                <Box>
                    <Heading size="sm" color="blue.800">驻地与外部地堡情报网</Heading>
                    <Text fontSize="xs" color="gray.600">Y{world.current_year} M{world.current_month} · 阶段 {world.progress_stage || 'arrival'} · 40 号派出，现驻 {world.station_silo?.name || '未知地堡'}</Text>
                </Box>
                <VStack align="end" gap={1}>
                    <Badge colorPalette={world.silo1_simulated ? 'green' : 'gray'}>
                        {world.station_silo ? 'STATION ACTIVE' : 'LEGACY WORLD'}
                    </Badge>
                    <Badge colorPalette={suspicion >= 0.7 ? 'red' : suspicion >= 0.4 ? 'orange' : 'blue'}>
                        特工怀疑度 {suspicion.toFixed(2)} / 1.00
                    </Badge>
                </VStack>
            </HStack>

            {counterintelActive && (
                <Box mb={3} p={3} bg="red.50" borderRadius="md" border="1px solid" borderColor="red.300">
                    <Text fontSize="sm" color="red.700" fontWeight="bold">反间谍已触发：每次升阶额外增加怀疑度 +0.08，并降低驻地合法性 -0.02。</Text>
                </Box>
            )}

            <VStack align="stretch" gap={3}>
                {(world.neighbors || []).map((neighbor: any) => (
                    <Box key={neighbor.silo_id} p={3} bg="white" borderRadius="md">
                        <HStack justify="space-between">
                            <Text fontWeight="bold">{neighbor.name}{neighbor.silo_id === world.station_silo_id ? ' · 当前驻地' : neighbor.name === '18 号地堡' ? ' · 正典观察，不可干预' : ''}</Text>
                            <Badge colorPalette={neighbor.status === 'normal' ? 'green' : neighbor.status === 'unrest' ? 'orange' : 'red'}>
                                {statusText[neighbor.status] || neighbor.status}
                            </Badge>
                        </HStack>
                        <Text fontSize="xs" color="gray.500">
                            信道：{neighbor.signal_source} · 最后确认：Y{Math.floor(neighbor.last_confirmed / 360) + 100} M{Math.floor((neighbor.last_confirmed % 360) / 30) + 1}
                        </Text>
                        {neighbor.rebellion_band && <Text fontSize="sm">叛乱风险：{neighbor.rebellion_band} · 人口规模：{neighbor.population_band}</Text>}
                    </Box>
                ))}

                {world.silo1_simulated && (
                    <Box p={3} bg="white" borderRadius="md">
                        <Text fontWeight="bold" mb={2}>1 号地堡部门构成</Text>
                        <HStack wrap="wrap" gap={1}>
                            {(world.silo1_departments || []).map((name: string) => <Badge key={name} colorPalette={name === 'Archives' ? 'purple' : 'gray'}>{name}</Badge>)}
                        </HStack>
                        <Text fontSize="sm" color="gray.600" mt={3}>
                            {world.silo1_last_report || '尚未截获部门信号。推进一个月以建立初始情报。'}
                        </Text>
                        <HStack justify="space-between" mt={3} pt={2} borderTop="1px solid" borderColor="gray.200">
                            <Text fontSize="sm" fontWeight="bold">反向追查暴露：{Math.round((world.silo1_exposure || 0) * 100)}% · 下一阈值 {nextCounterintelThreshold(world.silo1_exposure || 0)}</Text>
                            <Badge colorPalette={(world.silo1_counterintel_stage || 0) > 0 ? 'red' : 'gray'}>反间谍阶段 {world.silo1_counterintel_stage || 0}</Badge>
                        </HStack>
                        <VStack align="stretch" gap={2} mt={2}>
                            {(world.silo1_operations || []).slice(-6).reverse().map((operation: any) => (
                                <Box key={operation.id} p={2} bg="gray.50" borderRadius="sm">
                                    <HStack justify="space-between">
                                        <Text fontSize="sm" fontWeight="bold">#{operation.id} {operation.label}</Text>
                                        <Badge colorPalette={operation.status === 'PENDING' ? 'orange' : operation.status === 'RESOLVED' ? 'green' : 'red'}>{operationStatusText[operation.status] || operation.status}</Badge>
                                    </HStack>
                                    <Text fontSize="xs" color="gray.500">生效：Y{operation.due_year} M{operation.due_month} · 怀疑度 +{operation.trace_suspicion}</Text>
                                    <Text fontSize="xs" color="gray.600">{operation.resolution || operation.effect}</Text>
                                </Box>
                            ))}
                        </VStack>
                    </Box>
                )}

                {world.station_silo && (
                    <Box p={3} bg="white" borderRadius="md" border="1px solid" borderColor="purple.200">
                        <HStack justify="space-between">
                            <Text fontWeight="bold">驻地行动反馈 · {world.station_silo.name}</Text>
                            <Badge colorPalette={(world.station_counterintel_stage || 0) > 0 ? 'red' : 'gray'}>反间谍阶段 {world.station_counterintel_stage || 0}</Badge>
                        </HStack>
                        <Text fontSize="sm" fontWeight="bold" mt={2}>反向追查暴露：{Math.round((world.station_exposure || 0) * 100)}% · 下一阈值 {nextCounterintelThreshold(world.station_exposure || 0)}</Text>
                        <Text fontSize="sm" color="gray.600" mt={2}>
                            {world.station_last_report || '驻地的四类动作都会在这里给出实测依据。'}
                        </Text>
                        <VStack align="stretch" gap={2} mt={2}>
                            {(world.station_operations || []).slice(-6).reverse().map((operation: any) => (
                                <Box key={operation.id} p={2} bg="purple.50" borderRadius="sm">
                                    <HStack justify="space-between">
                                        <Text fontSize="sm" fontWeight="bold">#{operation.id} {operation.label}</Text>
                                        <Badge colorPalette={operation.status === 'PENDING' ? 'orange' : operation.status === 'RESOLVED' ? 'green' : 'red'}>{operationStatusText[operation.status] || operation.status}</Badge>
                                    </HStack>
                                    <Text fontSize="xs" color="gray.500">生效：Y{operation.due_year} M{operation.due_month} · 怀疑度 +{operation.trace_suspicion}</Text>
                                    <Text fontSize="xs" color="gray.600">{operation.resolution || operation.effect}</Text>
                                </Box>
                            ))}
                        </VStack>
                    </Box>
                )}
            </VStack>
        </Box>
    );
};
