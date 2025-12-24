from django.shortcuts import render

# Create your views here.
def home(request):
    return render(request, 'home.html')

def send_msg(request):
    if request.method == 'POST':
        user_email = request.POST.get('mail') # gets the value of the input field with name 'mail'
        recipientsList = request.POST.getlist('recipients')
        message = request.POST.get('message')
        print(f"User Email: {user_email}")
        print(f"Recipients: {recipientsList}")
        print(f"Message Content: {message}")
    return render(request, 'home.html')