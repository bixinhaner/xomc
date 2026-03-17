<%@ page contentType="text/html;charset=UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c" %>
<%@ taglib uri="http://java.sun.com/jsp/jstl/fmt" prefix="fmt" %>

<%-- 生产厂家名称 --%>
<%
	if (currOMCLicInfo.uiType == LicenseCons.UI_BAICELLS) {
%>
<c:set var="manufacturer" value="BaiCells"/>
<%
	} else if (currOMCLicInfo.uiType == LicenseCons.UI_RY) {
%>
<c:set var="manufacturer" value="RY"/>
<%
	} else if (currOMCLicInfo.uiType == LicenseCons.UI_RH) {
%>
<c:set var="manufacturer" value="RH"/>
<%
	} else if (currOMCLicInfo.uiType == LicenseCons.UI_TIANYI) {
%>
<c:set var="manufacturer" value="TIANYI"/>
<%
	} else if (currOMCLicInfo.uiType == LicenseCons.UI_FENGHUO) {   //武汉虹信 定制化显示烽火图标  
%>
<c:set var="manufacturer" value="FengHuo"/>
<%
	} else {
		//其他情况，均默认以BaiCells显示
%>
<c:set var="manufacturer" value="NoLogo"/>
<%
	}
%>
<%-- 皮肤 --%>
<c:set var="easyui_themes" value="metro"/>
<%-- 管理系统名称 --%>
<c:set var="system_name" value="Small Cell Management"/>