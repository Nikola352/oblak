from cars.cars import Car


class User:
    def __init__(self, name, car:Car):
        self._name = name
        self._car = car
    def greet(self):
        print(f"Hello {self._name}")
        print(f"You drive {self._car.to_string()}")