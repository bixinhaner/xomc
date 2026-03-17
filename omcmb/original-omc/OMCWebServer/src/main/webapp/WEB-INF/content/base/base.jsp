<%@page import="com.baicells.omc.busi.system.au.CurrOMCLicInfo"%>
<%@page import="com.baicells.omc.busi.utils.ApplicationProperties"%>
<%@page import="com.baicells.omc.busi.system.utils.UserUtil"%>
<%@page import="java.util.ResourceBundle"%>
<%@page import="java.util.Locale"%>
<%
    // Load Internationalization Resource package
    
    HttpSession sion = request.getSession();
    Locale locale = null;
    if (sion.getAttribute("language_code") != null && !"".equals(sion.getAttribute("language_code"))) {
        locale = new Locale(sion.getAttribute("language_code").toString());
    } else {
        locale = request.getLocale();
    }

    ResourceBundle rb = ResourceBundle.getBundle("messages", locale);
%>
