import React, { useRef, useState } from 'react';
import { Box, Button, Text, VStack, SimpleGrid, Badge } from '@chakra-ui/react';
import { PrizeWheel, type PrizeWheelHandle } from 'react-prize-wheel-advanced';

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
    const [selectedOption, setSelectedOption] = useState<SiloOption | null>(null);
    const [showManual, setShowManual] = useState(false);
    const wheelRef = useRef<PrizeWheelHandle>(null);

    const spin = () => {
        if (isSpinning) return;

        setIsSpinning(true);
        setSelectedOption(null);
        setShowManual(false);

        // Trigger the wheel library's spin animation (winner is picked internally at random)
        wheelRef.current?.click();
    };

    const handleFinished = (index: number) => {
        setIsSpinning(false);
        setSelectedOption(SILO_OPTIONS[index]);
    };

    const handleConfirm = () => {
        if (selectedOption) {
            onSelect(selectedOption.number);
        }
    };

    const handleManualSelect = (option: SiloOption) => {
        setSelectedOption(option);
    };

    return (
        <VStack gap={6} w="full">
            <PrizeWheel
                ref={wheelRef}
                segments={SILO_OPTIONS.map(o => String(o.number))}
                segColors={SILO_OPTIONS.map(o => o.color)}
                onFinished={handleFinished}
                size={300}
                primaryColor="#3182CE"
                contrastColor="#FFFFFF"
                buttonText="SPIN"
                spinDuration={4}
            />

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
