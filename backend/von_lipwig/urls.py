from django.urls import path
from . import views

urlpatterns = [
    path('', views.home, name='home'), #if the URL is von_lipwig/ then call views.home and take the user to the home page
    path('send_msg', views.send_msg, name='send_msg'), #if the URL is von_lipwig/send_msg then call views.send_msg to process the message sending
]