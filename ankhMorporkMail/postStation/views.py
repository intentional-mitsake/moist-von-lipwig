from django import views
from django.shortcuts import render

# Create your views here.

def post_station(request):
    return render(request, 'home.html')