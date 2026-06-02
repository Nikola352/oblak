class Car:
    def __init__(self, brand, model, year):
        self._brand = brand
        self._model = model
        self._year = year

    def to_string(self):
        return f"{self._brand}, {self._model}, {self._year}"