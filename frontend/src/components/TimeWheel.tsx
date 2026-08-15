import React, { useRef, useState } from 'react';
import { Box, Button, Text, VStack, SimpleGrid, Badge } from '@chakra-ui/react';
import { PrizeWheel, type PrizeWheelHandle } from 'react-prize-wheel-advanced';

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
    const [selectedOption, setSelectedOption] = useState<TimeOption | null>(null);
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
        setSelectedOption(TIME_OPTIONS[index]);
    };

    const handleConfirm = () => {
        if (selectedOption) {
            onSelect(selectedOption.year);
        }
    };

    const handleManualSelect = (option: TimeOption) => {
        setSelectedOption(option);
    };

    return (
        <VStack gap={6} w="full">
            <PrizeWheel
                ref={wheelRef}
                segments={TIME_OPTIONS.map(o => o.label)}
                segColors={TIME_OPTIONS.map(o => o.color)}
                onFinished={handleFinished}
                size={300}
                primaryColor="#3182CE"
                contrastColor="#FFFFFF"
                buttonText="SPIN"
                spinDuration={1}
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
