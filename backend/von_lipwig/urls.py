from django.urls import path
from . import views

urlpatterns = [
    path('', views.home, name='home'), #if the URL is von_lipwig/ then call views.home and take the user to the home page
]