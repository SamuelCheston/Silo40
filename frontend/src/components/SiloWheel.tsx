import React, { useState } from 'react';
import { Box, Button, Text, VStack, SimpleGrid, Badge } from '@chakra-ui/react';

interface SiloOption {
    label: string;
    number: number;
    color: string;
}

const SILO_OPTIONS: SiloOption[] = Array.from({ length: 50 }, (_, i) => i + 2)
    .filter(num => num !== 17 && num !== 18 && num !== 40)
    .map(num => ({
        label: `Silo ${num}`,
        number: num,
        color: `hsl(${(num * 137.5) % 360}, 70%, 50%)` // Generate distinct colors using golden angle
    }));

interface SiloWheelProps {
    onSelect: (number: number) => void;
}

export const SiloWheel: React.FC<SiloWheelProps> = ({ onSelect }) => {
    const [isSpinning, setIsSpinning] = useState(false);
    const [rotation, setRotation] = useState(0);
    const [selectedOption, setSelectedOption] = useState<SiloOption | null>(null);
    const [showManual, setShowManual] = useState(false);

    const spin = () => {
        if (isSpinning) return;

        setIsSpinning(true);
        setSelectedOption(null);
        setShowManual(false);

        const randomIndex = Math.floor(Math.random() * SILO_OPTIONS.length);
        const segmentAngle = 360 / SILO_OPTIONS.length;
        const targetRotation = 1800 + (randomIndex * segmentAngle) + (segmentAngle / 2);
        
        setRotation(prev => prev + targetRotation);

        setTimeout(() => {
            setIsSpinning(false);
            setSelectedOption(SILO_OPTIONS[randomIndex]);
        }, 3000);
    };

    const handleConfirm = () => {
        if (selectedOption) {
            onSelect(selectedOption.number);
        }
    };

    const handleManualSelect = (option: SiloOption) => {
        setSelectedOption(option);
        const index = SILO_OPTIONS.findIndex(o => o.number === option.number);
        const segmentAngle = 360 / SILO_OPTIONS.length;
        const targetRotation = (index * segmentAngle) + (segmentAngle / 2);
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
                    {SILO_OPTIONS.map((option, index) => {
                        const angle = 360 / SILO_OPTIONS.length;
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
                                    top="10px"
                                    left="50%"
                                    transform="translateX(-50%)"
                                    textAlign="center"
                                    w="80px"
                                    style={{ transformOrigin: '50% 140px' }}
                                >
                                    <Text fontSize="10px" fontWeight="bold" color="white" textShadow="0 0 2px rgba(0,0,0,0.8)">
                                        {option.number}
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
                        <Text fontSize="xl" fontWeight="bold" color="white">
                            {selectedOption.label}
                        </Text>
                    </Box>

                    <VStack gap={2} w="full">
                        <Button colorPalette="green" size="lg" onClick={handleConfirm} w="full">
                            Accept & Continue
                        </Button>
                        
                        {!showManual ? (
                            <Button variant="ghost" size="sm" onClick={() => setShowManual(true)}>
                                Not satisfied? Choose manually
                            </Button>
                        ) : (
                            <VStack w="full" gap={3} mt={2}>
                                <Text fontSize="sm" color="gray.400">Select your preferred Silo:</Text>
                                <SimpleGrid columns={4} gap={2} w="full" maxH="200px" overflowY="auto" p={2}>
                                    {SILO_OPTIONS.map((option) => (
                                        <Button
                                            key={option.number}
                                            variant={selectedOption.number === option.number ? "solid" : "outline"}
                                            colorPalette={selectedOption.number === option.number ? "blue" : "gray"}
                                            size="xs"
                                            onClick={() => handleManualSelect(option)}
                                        >
                                            {option.number}
                                        </Button>
                                    ))}
                                </SimpleGrid>
                            </VStack>
                        )}
                        
                        <Button variant="ghost" size="sm" onClick={spin} mt={2}>
                            Spin Again
                        </Button>
                    </VStack>
                </VStack>
            )}
        </VStack>
    );
};