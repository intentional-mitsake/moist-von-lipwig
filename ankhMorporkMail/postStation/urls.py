from django.urls import path
from . import views

urlpatterns = [
    path('', views.post_station, name='post_station')
]