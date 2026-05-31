from cars.cars import Car
from user import User


def handle(payload):
    print("I am free!")
    user = User("nikola", Car("Volkswagen","Polo N9", "2005"))
    user.greet()

