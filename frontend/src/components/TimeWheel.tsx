import React, { useState } from 'react';
import { Box, Button, Text, VStack, SimpleGrid, Badge } from '@chakra-ui/react';

interface TimeOption {
    label: string;
    year: number;
    color: string;
}

const TIME_OPTIONS: TimeOption[] = [
    { label: '10 years before', year: 112, color: '#E53E3E' }, // Red
    { label: '5 years before', year: 117, color: '#DD6B20' },  // Orange
    { label: 'Just now', year: 122, color: '#38A169' },       // Green
    { label: '5 years after', year: 127, color: '#3182CE' },  // Blue
    { label: '10 years after', year: 132, color: '#805AD5' }, // Purple
];

interface TimeWheelProps {
    onSelect: (year: number) => void;
}

export const TimeWheel: React.FC<TimeWheelProps> = ({ onSelect }) => {
    const [isSpinning, setIsSpinning] = useState(false);
    const [rotation, setRotation] = useState(0);
    const [selectedOption, setSelectedOption] = useState<TimeOption | null>(null);
    const [showManual, setShowManual] = useState(false);

    const spin = () => {
        if (isSpinning) return;

        setIsSpinning(true);
        setSelectedOption(null);
        setShowManual(false);

        const randomIndex = Math.floor(Math.random() * TIME_OPTIONS.length);
        const segmentAngle = 360 / TIME_OPTIONS.length;
        const targetRotation = 1800 + (randomIndex * segmentAngle) + (segmentAngle / 2);
        
        setRotation(prev => prev + targetRotation);

        setTimeout(() => {
            setIsSpinning(false);
            setSelectedOption(TIME_OPTIONS[randomIndex]);
        }, 3000);
    };

    const handleConfirm = () => {
        if (selectedOption) {
            onSelect(selectedOption.year);
        }
    };

    const handleManualSelect = (option: TimeOption) => {
        setSelectedOption(option);
        // Automatically scroll the wheel to that option for visual feedback
        const index = TIME_OPTIONS.findIndex(o => o.year === option.year);
        const segmentAngle = 360 / TIME_OPTIONS.length;
        const targetRotation = (index * segmentAngle) + (segmentAngle / 2);
        // We add enough full rotations to make it look like a smooth transition if needed,
        // but here we just set it to the base angle plus current full rotations.
        const fullRotations = Math.floor(rotation / 360) * 360;
        setRotation(fullRotations + targetRotation);
    };

    return (
        <VStack gap={6} w="full">
            <Box position="relative" w="300px" h="300px">
                {/* Pointer */}
                <Box
                    position="absolute"
                    top="-10px"
                    left="50%"
                    transform="translateX(-50%)"
                    zIndex={2}
                    w="0"
                    h="0"
                    borderLeft="15px solid transparent"
                    borderRight="15px solid transparent"
                    borderTop="30px solid white"
                />

                {/* The Wheel */}
                <Box
                    w="full"
                    h="full"
                    borderRadius="full"
                    position="relative"
                    overflow="hidden"
                    border="4px solid white"
                    transition="transform 3s cubic-bezier(0.15, 0, 0.15, 1)"
                    style={{ transform: `rotate(-${rotation}deg)` }}
                >
                    {TIME_OPTIONS.map((option, index) => {
                        const angle = 360 / TIME_OPTIONS.length;
                        const rotate = index * angle;
                        return (
                            <Box
                                key={option.label}
                                position="absolute"
                                top="0"
                                left="0"
                                w="full"
                                h="full"
                                style={{
                                    transform: `rotate(${rotate}deg)`,
                                    transformOrigin: '50% 50%',
                                }}
                            >
                                <Box
                                    position="absolute"
                                    top="0"
                                    left="50%"
                                    w="50%"
                                    h="50%"
                                    bg={option.color}
                                    style={{
                                        transform: `skewY(${90 - angle}deg)`,
                                        transformOrigin: '0% 100%',
                                    }}
                                />
                                <Box
                                    position="absolute"
                                    top="20px"
                                    left="50%"
                                    transform="translateX(-50%)"
                                    textAlign="center"
                                    w="100px"
                                    style={{ transformOrigin: '50% 130px' }}
                                >
                                    <Text fontSize="xs" fontWeight="bold" color="white" textShadow="0 0 4px rgba(0,0,0,0.5)">
                                        {option.label}
                                    </Text>
                                </Box>
                            </Box>
                        );
                    })}
                </Box>
            </Box>

            {!selectedOption && !isSpinning && (
                <Button
                    colorPalette="blue"
                    size="lg"
                    onClick={spin}
                    w="200px"
                >
                    Spin Wheel
                </Button>
            )}

            {isSpinning && (
                <Button colorPalette="blue" size="lg" loading loadingText="Spinning..." w="200px" disabled>
                    Spinning...
                </Button>
            )}

            {selectedOption && !isSpinning && (
                <VStack gap={4} w="full">
                    <Box textAlign="center" p={4} bg="gray.700" borderRadius="md" w="full">
                        <Badge colorPalette={showManual ? "orange" : "green"} mb={2}>
                            {showManual ? "Manual Overridden" : "Wheel Result"}
                        </Badge>
                        <Text fontSize="xl" fontWeight="bold">
                            {selectedOption.label}
                        </Text>
                        <Text fontSize="md" color="gray.400">
                            Year: {selectedOption.year}
                        </Text>
                    </Box>

                    <VStack gap={2} w="full">
                        <Button colorPalette="green" size="lg" onClick={handleConfirm} w="full">
                            Accept & Start Game
                        </Button>
                        
                        {!showManual ? (
                            <Button variant="ghost" size="sm" onClick={() => setShowManual(true)}>
                                Not satisfied? Choose manually
                            </Button>
                        ) : (
                            <VStack w="full" gap={3} mt={2}>
                                <Text fontSize="sm" color="gray.400">Select your preferred starting time:</Text>
                                <SimpleGrid columns={1} gap={2} w="full">
                                    {TIME_OPTIONS.map((option) => (
                                        <Button
                                            key={option.year}
                                            variant={selectedOption.year === option.year ? "solid" : "outline"}
                                            colorPalette={selectedOption.year === option.year ? "blue" : "gray"}
                                            size="sm"
                                            onClick={() => handleManualSelect(option)}
                                        >
                                            {option.label} (Year {option.year})
                                        </Button>
                                    ))}
                                </SimpleGrid>
                            </VStack>
                        )}
                        
                        <Button variant="ghost" size="sm" onClick={spin} mt={2}>
                            Spin Again
                        </Button>
                    </VStack>

                    <Text fontSize="xs" fontStyle="italic" textAlign="center" color="gray.500">
                        Juliette joined mechanical in Year 122.
                        <br />
                        (Born 109, Mother died at 13)
                    </Text>
                </VStack>
            )}
        </VStack>
    );
};

