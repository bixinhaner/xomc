<%@ page contentType="text/html;charset=UTF-8"
         import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%@page import="com.baicells.omc.busi.mts.utils.MtsComConstants" %>
<%@page import="com.baicells.omc.framework.utils.DateTimeUtil" %>
<%@ include file="/common/taglibs.jsp" %>
<%
    response.setHeader("Pragma", "No-cache");
    response.setHeader("Cache-Control", "no-cache");
    response.setDateHeader("Expires", 0);
    UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
    Boolean showLang = false;
%>
<!DOCTYPE html>

<html>
<head>
    <script type="text/javascript">
		var omctoken = '${token}',
			manufacturer = '${manufacturer}',
			offlineMapEnable = '${mapEnable}' == 'true'?true:false,
			offlineMapServer = '${mapUrl}',
			locationDetection = '${locationDetection}' == '1'?true:false,
			latitudeToleranceRange = '${latitudeToleranceRange}',
			enableWebPerformanceMode = '${enableWebPerformanceMode}' == 'true'? true:false; // 是否启用OMC web性能模式，启用则只保留当前tab的content节点，其他tab的content抹除

		/**
		* 在Safari和IE8上执行 new Date('2020-06-08 11:36:45'); 会得到Invalid Date
		* 本函数重写默认的Date函数，以解决其在Safari，IE8上的bug
	    **/
	    Date = function (Date) {
	      MyDate.prototype = Date.prototype;
	      MyDate.now = Date.now;
	      MyDate.getHours = Date.getHours;
	      MyDate.parse = Date.parse;
	      MyDate.getNow = function() {
	    	  return new MyDate(gloableTime);
	      }
	      return MyDate;

	      function MyDate() {
	        // 当只有一个参数并且参数类型是字符串时，把字符串中的-替换为/
	        if (arguments.length === 1) {
	          let arg = arguments[0];
	          if (Object.prototype.toString.call(arg) === '[object String]' && arg.indexOf('T') === -1) {
	            arguments[0] = arg.replace(/-/g, "/");
	          }
	        }
	        let bind = Function.bind;
	        let unbind = bind.bind(bind);
	        return new (unbind(Date, null).apply(null, arguments));
	      }
	    }(Date);

    	var timeZone = '${timeZone}';
		//服务器时间
		var gloableTime = '${init_operator_time}';
		//是否中文
		var isLocalZH = '${localeIsZh}' == 1;
	    //URL请求
	    var ctx = "${ctx}";
        var webRootPath = '${ctx}';
        var omcVersion = "${omcVersion}";

        //是否为QB版本  true/false
        var isQb = "${isQb}";
        //产品类型
        var isCloud = "${isCloud}";
        var isLocal = "${isLocal}";

        //支持的设备类型
        var isSupportCPE = "${isSupportCPE}";
        var isSupportEPC = "${isSupportEPC}";
        var isSupportHalob = "${isSupportHalob}";
        var isSupportUPS = "${isSupportUPS}";
        var isSupportEGW = "${isSupportEGW}";
        var isLWAEnable = "${lwaEnable}" == "true";

        //getMsg请求间隔时间
        var getMsgTime = "${quartzRefreshMsgSeconds}" ? "${quartzRefreshMsgSeconds}" + "000" : "6000";
        var getMsgTimeAlarm = "${quartzRefreshAlarmMsgSeconds}" ? "${quartzRefreshAlarmMsgSeconds}" + "000" : "6000";

        //是否是超级管理员
        var is_super_user = "<%=ui.isSuperAdmin()%>";
        //是否为运营商管理员
        var is_build_user = "<%=ui.isOperatorBuiltInAdmin()%>";
        //SAS功能是否打开  1-打开   0-关闭
        var SASEnble = '${sasEnable}';

      	//是否显示Site ID
        var siteIdShow = "${deviceSiteIdShow}";

		//是否显示AdditilnalCol
        var enbAdditionalColShow = "${enbAdditionalColShow}";
		// 北向运营商key
		var northOperatorScenario = "${northOperatorScenario}";
		// 判定赞比亚
		var scenarioKey = "${scenarioKey}";
        // 注册 显示Site ID
        var registerSiteIdShow = "${showSiteId}";
        // 不同北向运营商key显示不同
		var siteIdLabel = enbAdditionalColShow == 'true' ? northOperatorScenario == 'S0007' ? 'Site code' : 'Shop ID' : 'Shop ID';
		var siteNameLabel = enbAdditionalColShow == 'true' ? northOperatorScenario == 'S0007' ? 'eNodeB Name' : '<%=rb.getString("ZhanZhiMingCheng")%>' : '<%=rb.getString("ZhanZhiMingCheng")%>';
        // 不同北向运营商key S00016显示Site ID
        if(registerSiteIdShow == 'true'){
            siteIdLabel = 'Site ID'
        }
        //系统设置中CPE信号强度设置范围值
        var rsrp1 = "${rsrp_val0}";   //边界  小
        var rsrp2 = "${rsrp_val1}";   //边界  大

        //系统设置中UE信号强度设置范围值
        var uersrp1 = "${uersrp_val0}";   //边界  小
        var uersrp2 = "${uersrp_val1}";   //边界  大

		//获取批量操作权限
		var batchOperation = "${batchOperation}" == '1';

        var recycleBinEnable = "${recycleBinEnable}" == 'on';

        var multipartMaxFileSize = '${multipartMaxFileSize}';

		var user_code = unescape('<%=ui.getUsercode().replace("\'","%27").replace("\"","%22").replace("\\","%5C")%>');

        var operator_code = "<%=ui.getOperator_id()%>";
        var global_user_id = "<%=ui.getUid()%>";
        var CuoWuMa = "<%=rb.getString("CuoWuMa")%>";
        var CuoWuXinXi = "<%=rb.getString("CuoWuXinXi")%>";
        var ZhuangTaiShanChu = "<%=rb.getString("ZhuangTaiShanChu")%>";
        var ZhuangTaiJianLi = "<%=rb.getString("ZhuangTaiJianLi")%>";
        var ZhuangTaiTiJiaoDanMeiYingYong = "<%=rb.getString("ZhuangTaiTiJiaoDanMeiYingYong")%>";
        var BeiQiangZhiTuiChu = "<%=rb.getString("BeiQiangZhiTuiChu")%>";
        var ZuBeiShanChuYongHuTuiChu = "<%=rb.getString("ZuBeiShanChuYongHuTuiChu")%>"
        var HuoQuBuDaoCanShuZhi = "<%=rb.getString("HuoQuBuDaoCanShuZhi")%>";
        var XuLieHao = "<%=rb.getString("XuLieHao")%>";
        var MaoHao = "<%=rb.getString("MaoHao")%>";
        var KaiQi = "<%=rb.getString("KaiQi")%>";
        var GuanBi = "<%=rb.getString("GuanBi")%>";
        var sessionTimeOutMin = "${sessionTimeOutMin}";
        var TiShi = "<%=rb.getString("TiShi")%>";
        var SuoDingYongHuTiShi = "<%=rb.getString("SuoDingYongHuTiShi")%>";
        var ChongQiWanChengBingLianJieDaoOMC = "<%=rb.getString("ChongQiWanChengBingLianJieDaoOMC")%>";
        var HuiFuChuChangSheZhiWanChengBingLianJieDaoOMC = "<%=rb.getString("HuiFuChuChangSheZhiWanChengBingLianJieDaoOMC")%>";
        var XiaFaMingLingChaoShi="<%=rb.getString("XiaFaMingLingChaoShi")%>";
        var WuPeiZhiXinXi="<%=rb.getString("WuPeiZhiXinXi")%>";
        var YouJianZiShiYingKuanDu = "<%=rb.getString("YouJianZiShiYingKuanDu")%>";
        var TongBuZhong="<%=rb.getString("TongBuZhong")%>";
        var TongBuChengGongShiJian="<%=rb.getString("TongBuChengGongShiJian")%>";
        var TongBuShiBaiShiJian="<%=rb.getString("TongBuShiBaiShiJian")%>";
        var language = "<%=language%>";
        var TongBuZhong="<%=rb.getString("TongBuZhong")%>";
        var TongBuChengGongShiJian="<%=rb.getString("TongBuChengGongShiJian")%>";
        var TongBuShiBaiShiJian="<%=rb.getString("TongBuShiBaiShiJian")%>";
        var QingShuRuMiMa="<%=rb.getString("QingShuRuMiMa")%>";
        var ChongQiShengXiao="<%=rb.getString("ChongQiShengXiao")%>";
        var ShuangZaiBoXing = "<%= rb.getString("ShuangZaiBoXing")%>";
        var BiaoZhunXing = "<%= rb.getString("BiaoZhunXing")%>";
        var BuZaiXian = "<%= rb.getString("ShouYe_BuZaiXian")%>";
        var QueRen = "<%= rb.getString("QueRen")%>";
        var QueRenBaoCunBianGeng = "<%= rb.getString("QueRenBaoCunBianGeng")%>";
        var DengDai="<%= rb.getString("DengDai")%>";
        var JinXingZhong="<%= rb.getString("JinXingZhong")%>";
        var YiJieShu="<%= rb.getString("YiJieShu")%>";
        var ShiBai="<%= rb.getString("ShiBai")%>";
        var ChengGong="<%= rb.getString("ChengGong")%>";
        var BuFenChengGong="<%= rb.getString("BuFenChengGong")%>";
        var ZhengZaiZhongZhi="<%= rb.getString("ZhengZaiZhongZhi")%>";
        var ZhongZhi="<%= rb.getString("ZhongZhi")%>";
        var RuanJianBanBenXiangXiXinXi="<%= rb.getString("RuanJianBanBenXiangXiXinXi")%>";
        var ChaKan="<%=rb.getString("ChaKan")%>";
        var ZanTing="<%=rb.getString("ZanTing")%>";
        var CaoZuo = "<%=rb.getString("CaoZuo")%>";
        var showLang = "${showLang}";
        var QueRenYiChu = "<%=rb.getString("QueRenYiChu")%>";
        var XSSValidInfo = "<%=rb.getString("XSSValidInfo")%>";
        var operatorCodeGloab = "${SYSSESSIONKEY.operator_id}";
        var msgObj = {
        		ChangDuChaoChuFanWei : "<%=rb.getString("ChangDuChaoChuFanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%>",
        		ZiFu : "<%=rb.getString("ZiFu")%>",
        		phoneMessage : "<%=rb.getString("DianHuaGeShi")%>",
        		QingShuRuYouXiang : "<%=rb.getString("QingShuRu")%><%=rb.getString("YouXiang")%>",
        		CuoWuGeShi : "<%=rb.getString("CuoWuGeShi")%>",
        		ShenFenZhengGeShi : "<%=rb.getString("ShenFenZhengGeShi")%>"
        	}
        var validateMsg = JSON.stringify(msgObj);
		var JieShuShiJianBuNengXiaoYuKaiShiShiJian = "<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>";
		var QueDingQingKongSuoXuanSheBei = "<%=rb.getString("QueDingQingKongSuoXuanSheBei")%>";
		var QueDingQingKongYiXuanShuJu = "<%=rb.getString("QueDingQingKongYiXuanShuJu")%>";
		var QueDing = "<%=rb.getString("QueDing")%>";
		var QuXiao = "<%=rb.getString("QuXiao")%>";
		var ChuShiHuaZhong = "<%=rb.getString("ChuShiHuaZhong")%>";
		var YuanTongBuZhong = "<%=rb.getString("YuanTongBuZhong")%>";
		var YuanTongBuWanCheng = "<%=rb.getString("YuanTongBuWanCheng")%>";
		var distributedVue = '';
		// componets.js中默认值国际化
		var vueComponetsMsg = {
			  title: '<%=rb.getString("XinXi")%>',
			  name: '<%=rb.getString("MingCheng")%>',
			  ok: '<%=rb.getString("QueDing")%>',
			  cancel: '<%=rb.getString("QuXiao")%>',
			  clear: '<%=rb.getString("QingChu")%>',
			  reset: '<%=rb.getString("ChaXunChongZhi")%>',
			  other: '<%=rb.getString("QiTa")%>',
			  query: '<%=rb.getString("ChaXun")%>',
			  advanceQuery: '<%=rb.getString("GaoJiChaXun")%>',
			  error: '<%=rb.getString("CuoWu")%>',
			  exist: '<%=rb.getString("YiCunZai")%>',
			  selected: '<%=rb.getString("YiXuan")%>'
			}
    </script>

    <%
		if (currOMCLicInfo.uiType == LicenseCons.UI_BAICELLS) {
	%>
	<title>BaiOMC</title>
	<%
		} else if (currOMCLicInfo.uiType == LicenseCons.UI_RY) {
	%>
	<title>OMC</title>
	<%
		} else if (currOMCLicInfo.uiType == LicenseCons.UI_RH) {
	%>
	<title>Sunsea aiot</title>
	<%
		} else if (currOMCLicInfo.uiType == LicenseCons.UI_TIANYI) {
	%>
	<title>OMC</title>
	<%
		} else if (currOMCLicInfo.uiType == LicenseCons.UI_FENGHUO) {
	%>
	<title>HeMS</title>
	<%
		} else {
			//其他情况，均默认以OMC显示
	%>
	<title>OMC</title>
	<%
		}
	%>

    <link rel="stylesheet"type="text/css"  href="${ctx}/js/vxe-table/style.css?_=${omc_ver}">
    <link rel="stylesheet" type="text/css" href="${ctx}/js/jquery-easyui/themes/${easyui_themes}/easyui.css?_=${omc_ver}"/>
    <link rel="stylesheet" type="text/css" href="${ctx}/js/jquery-easyui/themes/icon.css?_=${omc_ver}"/>
    <link rel="stylesheet" type="text/css" href="${ctx}/css/normal.css?_=${omc_ver}"/>
    <link rel="stylesheet" type="text/css" href="${ctx}/css/element.css?_=${omc_ver}"/>
    <link rel="stylesheet" type="text/css" href="${ctx}/css/iconfont.css?_=${omc_ver}"/>
    <link rel="stylesheet" type="text/css" href="${ctx}/css/skin-common.css?_=${omc_ver}" />
    <link rel="stylesheet" type="text/css" href="${ctx}/css/style.css?_=${omc_ver}" />
    <link rel="stylesheet" type="text/css" href="${ctx}/skin/${manufacturer}/skin.css?_=${omc_ver}" />
    <link rel="Shortcut Icon" type="image/x-icon" href="${ctx}/favicon.ico" />
    <link rel="stylesheet" type="text/css" href="${ctx}/css/templateStyle.css?_=${omc_ver}" />
    <link rel="stylesheet" type="text/css" href="${ctx}/css/app.css?_=${omc_ver}" />
    <link rel="stylesheet" type="text/css" href="${ctx}/css/color.css?_=${omc_ver}" />
	<!-- OpenLayers CSS -->
	<link rel="stylesheet" href="${ctx}/js/ol/ol.css?_=${omc_ver}" type="text/css" />

	<!-- OpenLayers JavaScript Library -->
	<script type="text/javascript" src="${ctx}/js/ol/ol.js?_=${omc_ver}"></script>
	<!-- OpenLayers Topo Helper -->
	<script type="text/javascript" src="${ctx}/js/ol/ol-topo-helper.js?_=${omc_ver}"></script>

    <script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery-3.5.1.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/element/vue.min.js?_=${omc_ver}"></script>

    <!-- vxe-table 资源引入 -->
    <!-- 先加载 vxe-table 的依赖：VxeUI 核心库和 XEUtils 工具库（使用 CDN） -->
	<script type="text/javascript" src="${ctx}/js/vxe-table/xe-utils.umd.min.js?_=${omc_ver}"></script>
    <!-- 然后加载 vxe-table -->
	<script type="text/javascript" src="${ctx}/js/vxe-table/vxe-table.umd.min.js?_=${omc_ver}"></script>

    <script type="text/javascript" src="${ctx}/js/element/axios.min.js?_=${omc_ver}"></script>
    <c:if test="${localeIsZh == '1'}">
    	<script type="text/javascript" src="${ctx}/js/element/element.js?_=${omc_ver}"></script>
    </c:if>
    <c:if test="${localeIsZh != '1'}">
    	<script type="text/javascript" src="${ctx}/js/element/element-en.js?_=${omc_ver}"></script>
    </c:if>

	<script type="text/javascript" src="${ctx}/js/element/viewGrid.js?_=${omc_ver}"></script>

    <script type="text/javascript" src="${ctx}/js/element/components.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.serializejson.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/bi/home/browser.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/bi/home/jquery.dragsort.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.easyui.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/validate-extend.js?_=${omc_ver}"></script>
	<script type="text/javascript" src="${ctx}/js/jsencrypt.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/crypto-js.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/bi/home/main.js?_=${omc_ver}"></script>
    <c:if test="${localeIsZh == '1'}">
    	<script type="text/javascript" src="${ctx}/js/jquery-easyui/locale/easyui-lang-zh_CN.js?_=${omc_ver}"></script>
    </c:if>
    <script type="text/javascript" src="${ctx}/js/utils.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/date-utils.js?_=${omc_ver}"></script>
	<script type="text/javascript" src="${ctx}/js/echarts/echarts.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/datagrid-detailview.js"></script>
	<script type="text/javascript" src="${ctx}/js/bi/home/jquery.dragsort.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/bi/home/menu.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/easyui.pairgrid.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/easyui.query.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/jquery-easyui/easyui.gridbox.js?_=${omc_ver}"></script>
	<script type="text/javascript" src="${ctx}/js/node-forge/forge.min.js?_=${omc_ver}"></script>

	<script type="text/javascript" src="${ctx}/js/go.js?_=${omc_ver}"></script>
	<script type="text/javascript" src="${ctx}/js/vis-network.min.js?_=${omc_ver}"></script>

	<script type="text/javascript" src="${ctx}/js/sortTable/Sortable.min.js?_=${omc_ver}"></script>

    <script>
		var eventAllBus = new Vue();
		$.fn.pairgrid.defaults.selectedMsg = '<%=rb.getString("YiJingXuanZe")%>';
		$.fn.pairgrid.defaults.messages.tips = '<%=rb.getString("QueRen")%>';
		$.fn.pairgrid.defaults.messages.delete = '<%=rb.getString("QueDingQingKongYiXuanShuJu")%>';
    </script>
	<style>
		/* 修复vxe-table导致el-table fixed的高度计算问题 */
        .el-table .el-table__fixed,
        .el-table .el-table__fixed-right {
            height: calc(100% - 9px) !important;
            bottom: auto !important;
        }
        .el-table--scrollable-x .el-table__fixed,
        .el-table--scrollable-x .el-table__fixed-right {
            height: calc(100% - 9px) !important;
            bottom: auto !important;
        }
        /* vxe-table 单元格溢出检测优化 */
        #tableHomeCellList .vxe-body--column .vxe-cell {
            padding-left: 6px !important;
            padding-right: 6px !important;
            max-width: 100% !important;
            box-sizing: border-box !important;
        }
        /* 确保单元格内的文本容器正确触发溢出 */
        #tableHomeCellList .vxe-body--column .vxe-cell .vxe-cell--label {
            max-width: 100% !important;
            display: block !important;
            overflow: hidden !important;
            text-overflow: ellipsis !important;
            white-space: nowrap !important;
        }
        /* vxe-table tooltip 白色背景样式 */
        .vxe-table--tooltip-wrapper,
        .vxe-table--tooltip-wrapper.theme--light {
            background-color: #ffffff !important;
            color: #333333 !important;
            border: 1px solid #dcdfe6 !important;
            box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1) !important;
            border-radius: 4px !important;
            max-width: 400px !important;
            word-break: break-all !important;
        }
        .vxe-table--tooltip-wrapper .vxe-table--tooltip-content,
        .vxe-table--tooltip-wrapper.theme--light .vxe-table--tooltip-content {
            color: #333333 !important;
        }
        .vxe-table--render-default .vxe-cell--checkbox .vxe-checkbox--icon {
            font-weight: 500;
        }
        table {
            font-size: 12px;
        }

		.imgIcon{
			font-size:18px;
			margin-right:10px;
		}
		.headerAlarmCon .bgCircle{
			display:inline-block;
			width:12px;
			height:12px;
			border-radius:50%;
			margin-left:39px;
		}
		.headerAlarmCon{
			cursor:pointer;
		}
		#sysMain .el-dropdown:hover{
			background:unset;
		}
		@media screen and (max-width:1280px) {
			.alarmTitle{
				display:none;
			}
		}
		.colorWhite{
			color:#ffffff;
		}
		.colorHui{
			color:#B0C9FF;
		}
		.popoverStyle{
			max-height: 430px;
			max-width: 440px;
			display:flex;
			flex-wrap: wrap;
			flex-direction: column;
		}
		.menuContent{
			width: 220px;
			font-size: 14px;
		}
		.menuContent .menuTitle{
			height: 40px;
			font-size: 14px;
			display: flex;
			align-items: center;
			font-weight: 550;
			line-height: 40px;
			box-sizing: border-box;
		}
		.menuContent .menuTitleOne{
			height: 40px;
			display: flex;
			align-items: center;
			font-size: 14px;
			font-weight: 550;
			line-height: 40px!important;
			box-sizing: border-box;
			cursor:pointer;
		}
		.menuTitleOneMain{
			border-left: 3px solid ;
			height: 14px;
			font-size: 14px;
			line-height: 14px;
			margin-left: 18px;
			padding-left: 18px;
		}
		.menuContent .menuTitleOne:hover div span{
			color: #FFF;
			border-bottom: 1px solid #FFF;
		}
		.menuContent .menuTitleOneSelect{
			height: 40px;
			display: flex;
			align-items: center;
			font-size: 14px;
			color: #FFF;
			font-weight: 550;
			line-height: 40px!important;
			box-sizing: border-box;
			cursor:pointer;
		}
		.menuTitleOneSelect .menuTitleOneMain{
			border-left: 3px solid ;
			height: 14px;
			font-size: 14px;
			font-weight: 550;
			line-height: 14px;
			margin-left: 18px;
			padding-left: 18px;
		}
		.menuTitleOneSelect .menuTitleOneMain span{
			border-bottom: 1px solid #FFF;
		}
		.menuPopoverClass .popper__arrow::after{
			border-right-color: #EBF1FF!important;
			left: 0!important;
		}

		.menuChildrenStyle{
			height: 30px;
			font-size: 12px;
			padding-left: 50px;
			font-weight: 550;
			line-height: 30px;
			cursor:pointer;
		}

		.menuChildrenStyle:hover span{
			color: #FFF;
			border-bottom: 1px solid #FFF;
		}
		.menuChildrenSelect{
			color: #FFF;
			height: 30px;
			font-weight: 550;
			padding-left: 50px;
			line-height: 30px;
			cursor:pointer;
		}
		.menuChildrenSelect span{
			border-bottom: 1px solid #FFF;
			font-size: 12px;
		}
		 .menuStyle .el-tooltip{
			padding: 0px 12px 0px 12px!important;
		}

		.menuPopoverClass{
			z-index: 9999!important;
			margin-left: 0px !important;
			padding: 5px 0px 5px 0px !important;
			border-radius:0px 4px 4px 0px;
		}
		#menuAnimate .el-menu-item.is-active:before{
			width: 0px!important;
		}
		#menuAnimate .el-menu-item.is-active:before{
			background: #EBF1FF;
		}
		#menuAnimate .logoTitle .el-icon-logo-omc:before{
			font-size: 44px;
		}
		.sysHeader .el-icon-logo-baiomc:before{
			content: "";
		}
		#menuAnimate .logoTitle .versionTitle{
			margin:26px 15px 0px 10px;
		}

		#menuAnimate .el-menu--collapse{
			width: 53px;
			background:#070b4b;
		}
		.el-icon-logo-omc:before{
			content:""
		}
		.collapseTitle{
			height:40px;
			line-height:40px;
			border-bottom:1px solid ;
		}
		.el-popper .popper__arrow{
			display:none!important;
		}

		.versionStyle{
			height:40px;
			line-height:40px;
			border-top:1px solid #E9E9E9;
			background:#070b4b;
			color:#fff;
		}
		.versionCollapseStyle{
			text-align:center;
		}
		#changeOperator .el-dialog{
			height: 88%;
		}
		#changeOperator .el-dialog__body{
			padding: 0px;
			margin-top: -40px;
		}
		#changeOperator .el-dialog__headerbtn{
			z-index: 9999
		}
		.curr_operator_div [class*="el-icon-menu"]:before {
			font-size: 16px!important;
			color: #7A7992;
		}
		.curr_operator_div_white [class*="el-icon-menu"]:before {
			color: #A2A2A2;
			font-size: 16px!important;
		}
		.sysHeader .curr_operator_div:hover [class*="el-icon-menu"]:before{
			color: #FFF!important;
		}

		.changeOperatorDiv{
			cursor: pointer;
		}
		.el-icon-menu-5G:before {
			font-size: 26px !important;
		}
		.deviceMigrationBox, .messageTitleWarp{
			width: 40px;
			height: 36px;
			display: flex;
			align-items: center;
			justify-content: center;
			border-left: 1px solid #FFFFFF;
		}
		.deviceMigrationBox_White, .messageTitleWarp_White{
			width: 40px;
			height: 36px;
			display: flex;
			align-items: center;
			justify-content: center;
			border-left: 1px solid #E9E9E9;
		}
		.deviceMigrationBox .el-icon::before, .messageTitleWarp .el-icon::before{
			font-size: 18px!important;
		}
		.deviceMigrationBox_White .el-icon::before, .messageTitleWarp_White .el-icon::before{
			color: #A2A2A2;
			font-size: 18px!important;
		}

		.circleTips { font-size: 12px; display: block; width: 5px; height: 5px; border-radius: 50%; background: red; position: absolute; top: 10px; right: 6px;}
		.inprogressTip { position: absolute; bottom: 0; left: 8px; }
		.messageCenterWarp { width: 360px; height: calc(100% - 30px); border-radius: 10px; background: #FFFFFF; position: absolute; top: 15px; right: 8px; z-index: 2000; border: 1px solid #D5DCEC;}
		.messageCenterWarp .headerWarp { height: 50px; width: 100%; line-height: 50px; border-bottom: 1px solid #D5DCEC; display: flex; justify-content: space-between; }
		.messageCenterWarp .batchDeleteBtn { font-size: 14px;margin-right: 12px; margin-top: 18px;}
		.messageCenterWarp .el-icon-operation-time:before,
		.messageCenterWarp .operIconBtn:before { color: #7A7992; }
		.messageCenterWarp .msgBox { height: calc(100% - 56px); margin: 0 8px; }
		.messageCenterWarp .msgBox .messageWarp:first-child .messageBox { border-top: none; }
		.messageCenterWarp .messageBox { border-top: 1px solid #D5DCEC; padding: 10px 10px; position: relative; }
		.messageCenterWarp .messageBox:hover,
		.messageBox:active { background: #F6F7FB; }
		.messageCenterWarp .messageBox:hover .deleteBtnShow { display: block !important; }
		.messageCenterWarp .msgBox .el-ctable .el-table__header-wrapper { display: none; }
		.messageCenterWarp .msgBox .el-table tr, .msgBox .el-table td { background: #FFFFFF !important; padding: 0;}
		.messageCenterWarp .msgBox .el-table .cell { padding: 0 !important; }
		.messageCenterWarp .msgBox .el-table--border { border: none; }
		.messageCenterWarp .msgBox .el-table td { border-bottom: 1px solid #D5DCEC;}
		.messageCenterWarp .deleteBtnShow,
		.messageCenterWarp .specialDeleteBtnShow { display: none; font-size: 14px; color:var(--main-color); position: absolute; top: 14px; right: 4px; }
		.messageCenterWarp .singleDeleteBtn:before { color:var(--main-color); }
		.messageCenterWarp .msgSuccessText { background: #F3FBEE; font-size: 12px; border-radius: 20px; padding: 0 12px; color: #4ED76E; height: 20px; line-height: 20px; }
		.messageCenterWarp .msgFailText { background: #FEEDE6; font-size: 12px; border-radius: 20px; padding: 0 12px; color:var(--main-color); height: 20px; line-height: 20px; }
		.messageCenterWarp .msgProgressText { background: #F1F5FF; font-size: 12px; border-radius: 20px; padding: 0 12px; color: #4D84FF; height: 20px; line-height: 20px; }

		.net-headerBoxCls {
	        height: 24px;
	        width: 100%;

	        display: flex;
	        align-items: center;
	        justify-content: center;
	        padding-top: 10px;
	    }
	    .net-headerBoxCls .el-icon::before{
	        font-size: 14px;
	        margin-right:5px;
	        color:rgba(0,0,0,0.8);
	    }
	    .net-headerBoxCls .commonRadioButton .el-radio-button__inner{
	        display: flex;
	        align-items: center
	    }
	    .net-headerBoxCls .el-radio-button.is-active .el-icon::before{
	        color:var(--main-color);
	    }
	    .el-menu--collapse .el-menu-item {
	    	padding:0 !important;
	    }
	    .menuStyle .el-tooltip {
	    	position:unset !important;
	    	padding:0 !important;
	    	width:32px !important;
	    	height:32px !important;
	    	border-radius:10px;
	    	text-align:center;
	    }
	    .menuStyle .el-tooltip .el-icon {
	    	line-height:32px;
	    	margin:0;
	    }
	    .el-menu-item  .el-icon {
	    	margin-right:15px;
	    	display:inline-block;
	    	width:22px;
	    }
		.alarmAlert {
			border-right: 1px solid gray;
			padding-right: 20px;
			height: fit-content;
			margin-top: 7px;
		}
		.alarmAlert .el-icon::before {
			color: unset;
		}
	</style>
</head>

<body class="darkblue loading">
<div id="sysMain" style="height:100%;">
	<audio id="alarmMusic" src="${ctx}/audio/alarm2.mp3"></audio>
	<div class='lockIconItem' v-if="showBrowserTip">
		<div style="margin:auto;display:flex">
			<p style='line-height:40px;'><span class="el-icon el-icon-circle-warning" style="margin-right:10px;font-size:16px;"></span><span style="font-size:14px;">您在用的浏览器可能与系统存在兼容问题，为了更好的提供产品服务，推荐您使用如下浏览器:</span></p>
			<div style="display:inline-block;margin-left:10px">
				<span class="browser_icon google_icon"></span>
				<span class="browser_icon safari_icon"></span>
				<span class="browser_icon firefox_icon"></span>
				<span class="browser_icon edge_icon"></span>
			</div>
		</div>
	</div>
	<el-container style="height:100%;">
		<el-header height="50px" style="display:flex;align-items:center;justify-content:space-between" :class="headerClass">
			<div class="header-left" style="display:flex;align-items:center;font-size:14px;">
				<div :class="logoCollapseClass"></div>
				<span  :class="muneToggleIcon" style="margin-right:10px;cursor:pointer;font-size:16px;" @click="menuCollapseClick"></span>
				<span v-if="false" class="path" v-for="(item,index) in headPath" :key="index">
					<span :class="(index == headPath.length-1) ? colorHighLight : '' " style="">{{item}}</span><span v-if="!(index == headPath.length-1)" style="margin-right:4px;">/</span>
				</span>
					<div class="net-headerBoxCls">
					<el-radio-group size="mini" v-model='headType' class="commonRadioButton" @change="headTypeChange" style='margin: 0 10px 10px;'>
		                <el-radio-button v-for="item in headBtnData" :label="item.code"><span :class="item.icon"></span>{{item.label}}</el-radio-button>
		            </el-radio-group>
	            </div>
			</div>

			<div class="header-right alarmPromptDiv" style="display:flex;">
				<!-- 北向告警接口 -->
				<!-- v-if="noItfnAlarmBtn == '0'" -->
				<div class="alarmAlert">
					<el-badge is-dot :hidden="unreadFlag">
						<span @click="showUnreadAlarm" class="el-icon el-icon-menu-alarm"></span>
					</el-badge>
				</div>
				<div v-if="noItfnAlarmBtn == '0'" onclick="showItfnAlarm();" class="alarm itfn_alarm" style="margin-right:25px;"><%=rb.getString("BeiXiangJieKouCuoWu")%></div>
				<div class="headerAlarm CODE_ALARM_VIEW hidden visible" style="display:flex;align-items:center;font-size:12px;" >
					<div class="headerAlarmCon criticalAlarm" @click="showCurrAliveAlarm('31001')">
						<span title="Critical" class="bgCircle" style="margin-left:0px;margin-right:4px;"></span><span class="alarmTitle">Critical</span>
						<span class="sub-mini-hidden" id="spanCriticalCount" style="margin-left:10px;">0</span>
					</div>
					<div class="headerAlarmCon majorAlarm" @click="showCurrAliveAlarm('31002')">
						<span title="Major" class="bgCircle" style="margin-left:30px;margin-right:4px;"></span><span class="alarmTitle">Major</span>
						<span class="sub-mini-hidden" id="spanMajorCount" style="margin-left:10px;">0</span>
					</div>
					<div class="headerAlarmCon minorAlarm" @click="showCurrAliveAlarm('31003')">
						<span title="Minor"  class="bgCircle" style="margin-left:30px;margin-right:4px;"></span><span class="alarmTitle">Minor</span>
						<span class="sub-mini-hidden" id="spanMinorCount" style="margin-left:10px;">0</span>
					</div>
					<div class="headerAlarmCon warningAlarm" @click="showCurrAliveAlarm('31004')">
						<span title="Warning"  class="bgCircle" style="margin-left:30px;margin-right:4px;"></span><span class="alarmTitle">Warning</span>
						<span class="sub-mini-hidden" id="spanWarningCount" style="margin-left:10px;margin-right:14px;">0</span>
					</div>
				</div>
				<!--消息中心-->
				<div :class="headerType == 'false'?'messageTitleWarp' : 'messageTitleWarp_White'" style='position: relative;'>
					<el-tooltip content='<%=rb.getString("XiaoXiZhongXin")%>' placement='bottom' :enterable=false>
						<span class="el-icon el-icon-circle-alarm" id='messageCenterBtn'></span>
					</el-tooltip>
					<span class='circleTips' v-show='circleTipShowOrHide'></span>
					<span class='inprogressTip' v-show='inprogressTipShowOrHide'></span>
				</div>
				<!--设备迁移-->
				<div :class="headerType == 'false'?'deviceMigrationBox' : 'deviceMigrationBox_White'" v-show="migrateRole == 'commercial'">
					<el-tooltip content='<%=rb.getString("SheBeiQianYi")%>' placement='bottom' :enterable=false>
						<span class="el-icon el-icon-move " @click="deviceMigrationClick"></span>
					</el-tooltip>

				</div>
				<div id="curr_operator_div" :class="headerType == 'false'?'curr_operator_div' : 'curr_operator_div_white'" style="display:flex;align-items: center;"  >
					<div @click="changeOperator" v-if="isCloud == 'true' && (isSuperAdmin == '1' || isDistributor == '1')" class="changeOperatorDiv" style="display:flex;align-items: center;">
						<span class="el-icon el-icon-menu-operator" style="margin-left:5px;"></span>
					</div>
					<div class="mini-hidden">
						<span id="curr_operator_span" style="margin:0px 5px 0px 5px;"></span>
						<span id="curr_time_span" style="display:-webkit-inline-box;">${init_operator_time}</span>
					</div>

				</div>

				<div class="omc_userOpList">
					<el-dropdown v-if="headerType == 'false'" style="font-size:12px;" id="dropdownSelect" trigger="click" @command="commandHandle"><!--  -->
						<span style="cursor:pointer;" class="el-dropdown-link">
							<%=rb.getString("HuanYingNin")%> ,<span><%=ui.getUsercode()%></span><i style="margin-left:5px;" class="el-icon-arrow-down"></i>
						</span>
						<el-dropdown-menu slot="dropdown">
							<el-dropdown-item command="changePassWord"><i class="imgIcon el-icon el-icon-operation-change"></i><%=rb.getString("XiuGaiMiMaOpt")%></el-dropdown-item>
							<!--<el-dropdown-item v-if="isCloud == 'true'" command="changeOperator"><i class="imgIcon el-icon el-icon-menu-operator"></i><%=rb.getString("YunYingShang")%></el-dropdown-item>-->
							<el-dropdown-item command="lockScreen"><i class="imgIcon el-icon el-icon-common-lock"></i><%=rb.getString("SuoPing")%></el-dropdown-item>
							<el-dropdown-item command="logout"><i class="imgIcon el-icon el-icon-common-logout"></i><%=rb.getString("TuiChu")%></el-dropdown-item>
						</el-dropdown-menu>
					</el-dropdown>
				</div>

			</div>
		</el-header>

		<el-container style="overflow: auto;">
			<el-aside id="menuAnimate" :style="menuContStyle" style="overflow:hidden;display:flex;flex-direction:column" ><!-- :id="'menuAnimate'" -->
				<el-menu ref="menuList" :default-active="defaultActive" :collapse="isCollapse" @open="menuOpen" @close="menuClose" unique-opened="true" @select="menuClick"  style="flex:auto;overflow:auto;" class="omcMenu">
					<template v-for="(val,name,index) in menuData" v-if="isCollapse"><!-- 菜单收起状态  -->
						<el-popover placement="right-start"  trigger="hover" v-if="val.children" ref="menuPopover" @show="popoverShow(val.id)"  @hide="popoverHide(val.id)"  class="menuStyle"  popper-class="menuPopoverClass">
							<div class='collapseTitle' :style="popoverWidth">
								<span style='margin-left:18px;'>{{val.menu_name}}</span>
							</div>
							<div  class="popoverStyle" :style="popoverWidth">
								<div v-for="(menu,key) in val.children">
									<div class="menuContent" >
										<div :id="'menu_'+menu.id" :index="menu.id" :url="menu.menu_url" :path="menu.path" v-if="!menu.children ||menu.menu_leaf == '1' " :poper-class="2" :class="[menuChildId == menu.id ?'menuTitleOneSelect':'menuTitleOne']" @click="menuChildrenClick(menu,val)" :fatherId="val.id"><div class="menuTitleOneMain"><span>{{menu.menu_name}}</span></div></div>
										<div class="menuTitle" v-if="menu.children&&menu.menu_leaf == '0'"><div class="menuTitleOneMain">{{menu.menu_name}}</div></div>
										<div v-if="menu.menu_leaf == '0'"  >
											<div v-for="(menuSub,key) in menu.children" :id="'menu_'+menuSub.id" :index="menuSub.id" :url="menuSub.menu_url" :path="menuSub.path" :poper-class="2" :class="[menuChildId == menuSub.id ?'menuChildrenSelect':'menuChildrenStyle']" @click="menuChildrenClick(menuSub,val)" :fatherId="val.id" >
												<span>{{menuSub.menu_name}}</span>
											</div>
										</div>
									</div>

								</div>
							</div>
							<div slot="reference">
								<el-menu-item :id="'menu_'+val.id" :index="val.id" :url="val.menu_url" :path="val.path" :isOne='1' :fatherId="val.id">
									<i :class="val.menu_icon"></i>
									<span slot="title">{{val.menu_name}}</span>
								</el-menu-item>
							</div>
						</el-popover>
						<div v-if="!val.children">
							<el-menu-item :id="'menu_'+val.id" :poper-class="1" :index="val.id" :url="val.menu_url" :path="val.path" :isOne='2' :fatherId="val.id" class="menuStyle">
								<i :class="val.menu_icon"></i>
								<span slot="title">{{val.menu_name}}</span>
							</el-menu-item>
						</div>
					</template>
					<template v-for="(val,name,index) in menuData" v-if="!isCollapse"><!-- 菜单展开状态  -->

						<el-popover placement="right-start"  trigger="hover" v-if="val.children" ref="menuPopover" @show="popoverShow(val.id)"  @hide="popoverHide(val.id)" popper-class="menuPopoverClass" >
							<div  class="popoverStyle" :style="popoverWidth">
								<div v-for="(menu,key) in val.children">
									<div class="menuContent" >
										<div :id="'menu_'+menu.id" :index="menu.id" :url="menu.menu_url" :path="menu.path" v-if="!menu.children ||menu.menu_leaf == '1' " :poper-class="2" :class="[menuChildId == menu.id ?'menuTitleOneSelect':'menuTitleOne']" @click="menuChildrenClick(menu,val)" :fatherId="val.id"><div class="menuTitleOneMain"><span>{{menu.menu_name}}</span></div></div>
										<div class="menuTitle" v-if="menu.children&&menu.menu_leaf == '0'"><div class="menuTitleOneMain">{{menu.menu_name}}</div></div>
										<div v-if="menu.menu_leaf == '0'" class="menuChild" >
											<div v-for="(menuSub,key) in menu.children" :id="'menu_'+menuSub.id" :index="menuSub.id" :url="menuSub.menu_url" :path="menuSub.path" :poper-class="2" :class="[menuChildId == menuSub.id ?'menuChildrenSelect':'menuChildrenStyle']" @click="menuChildrenClick(menuSub,val)" :fatherId="val.id" >
												<span>{{menuSub.menu_name}}</span>
											</div>
										</div>
									</div>

								</div>
							</div>
							<div slot="reference">
								<el-submenu :id="'menu_'+val.id" :index="val.id" v-if="val.menu_leaf == '0'" show-timeout="100" hide-timeout="100">
									<template slot="title">
										<i :class="val.menu_icon"></i>
										<span slot="title">{{val.menu_name}}</span>
									</template>
								</el-submenu>
								<el-menu-item :id="'menu_'+val.id" :index="val.id" :url="val.menu_url" :path="val.path" v-if="val.menu_leaf == '1'">
									<i :class="val.menu_icon"></i>
									<span slot="title">{{val.menu_name}}</span>
								</el-menu-item>
							</div>
						</el-popover>
						<div v-if="!val.children">
							<el-submenu :id="'menu_'+val.id" :index="val.id" v-if="val.menu_leaf == '0'" show-timeout="100" hide-timeout="100" >
								<template slot="title">
									<i :class="val.menu_icon"></i>
									<span slot="title">{{val.menu_name}}</span>
								</template>
							</el-submenu>
							<el-menu-item :id="'menu_'+val.id" :poper-class="1" :index="val.id" :url="val.menu_url" :path="val.path"  v-if="val.menu_leaf == '1'" >
								<i :class="val.menu_icon"></i>
								<span slot="title">{{val.menu_name}}</span>
							</el-menu-item>
						</div>
					</template>
				</el-menu>
				<div class="versionStyle" :class='versionSpan'>
					<span v-show="showVer" style='margin-left:10px;'><%=rb.getString("BanBenHao")%></span>
					<span>${omc_ver}</span>
				</div>
			</el-aside>

			<el-container id="omc_app_ctn" style="position: relative;flex-direction:column;background: #F6F7FB;">
				<el-navmenu v-if="menus.length>0 && !['2000','3000','5000','7000'].includes(defaultActive+'')" :default-active="menuDefaultActive" :list="menus" @select="menuSelect"></el-navmenu>

				<el-navigator v-if="navTabsEnable" ref="nav" @tab-removed="onTabRemoved" style="margin-top: 3px;"></el-navigator>

				<el-main style="position:relative;" v-show="!navTabsEnable">
					<div id="mainpage" style="position:absolute;overflow:hidden;top:8px;right:8px;bottom:8px;left:8px;"></div>
				</el-main>

				<!-- 消息中心 -->
				<div id='messsgeCenterBox' class='messageCenterWarp' v-show='messsgeCenterShowOrHide'>
					<transition name='el-zoom-in-right'>
						<div>
							<div class='headerWarp'>
								<div class='commonText14' style='margin-left: 20px;'><%=rb.getString("XiaoXiZhongXin")%></div>
								<div style='margin-right: 20px;' class='commonFlex'>
									<span @click='batchDeleteBtnClick' v-show='curMsgData.length > 0' class='el-icon el-icon-operation-delete operIconBtn batchDeleteBtn'></span>
									<span @click='closeMessageCenter' class='el-icon el-icon-close operIconBtn' style='margin-top: 18px; font-size: 14px;'></span>
								</div>
							</div>
							<div class='msgBox' id='messageResulBox'>
								<el-ctable id="msgTable" ref="msgTable" height="100%" :url='msgUrl' :query-params="msgTable_params" :time="6" :row-key="'id'" @load-success="msgTableLoadSuccess" :showHeader="false" :rownumber="false" :show-pager="false" :pagination="true">
									<el-table-column prop='smallCellCode'>
										<template slot-scope="scope">
											<div class='messageWarp'>
												<div class='messageBox'>
														<span class='commonGeneral12' style='white-space: pre-wrap; word-wrap: break-word; display: block; margin-right: 12px;font-weight: bold; '>SN: {{scope.row.smallCellCode}}, Cell Name: {{scope.row.cellName}}</span>
														<span v-if='scope.row.msgStatus == "IN_PROGRESS"' class='specialDeleteBtnShow el-icon el-icon-operation-delete singleDeleteBtn'></span>
														<span @click='singleDeleteBtnClick(scope.row.id)' v-if='scope.row.msgStatus != "IN_PROGRESS"' class='deleteBtnShow el-icon el-icon-operation-delete singleDeleteBtn'></span>
														<span class='commonTemplateText12' style='padding-top: 6px; white-space: pre-wrap; word-wrap: break-word; display: block;line-height: 18px;'><%=rb.getString("CaoZuo")%>: {{scope.row.msgType}}</span>
														<span class='commonTemplateText12' v-show='scope.row.msgStatus == "FAILURE"' style='padding-top: 6px; display: block; white-space: pre-wrap; word-wrap: break-word; line-height: 18px;'><%=rb.getString("ShiBaiYuanYin")%>: {{scope.row.msgContent}}</span>
														<div style='display: flex;padding-top: 10px;justify-content: space-between;'>
															<div class='msgSuccessText' v-show='scope.row.msgStatus == "SUCCESS"'><%=rb.getString("ChengGong")%></div>
															<div class='msgFailText' v-show='scope.row.msgStatus == "FAILURE"'><%=rb.getString("ShiBai")%></div>
															<div class='msgProgressText' v-show='scope.row.msgStatus == "IN_PROGRESS"'><%=rb.getString("JinXingZhong")%></div>
															<div class='commonTemplateText12 commonFlex' style='margin-right: 10px;'>
																<i class='el-icon el-icon-operation-time' style='margin: 5px 6px 0; font-size: 14px;'></i>
																<span>{{scope.row.msgTime}}</span>
															</div>
														</div>
													</div>
												</div>
										</template>
									</el-table-column>
								</el-ctable>
							</div>
						</div>
					</transition>

				</div>
			</el-container>
		</el-container>
	</el-container>
	<!--运营商弹窗页面部分 -->
	<el-dialog id="changeOperator"  :title="'<%=rb.getString("TianJia")%>'" top="5vh" :visible.sync="showChangeOperator" ref="changeOperatorDialog" :width="operatorDialogWidth"
		:close-on-click-modal="false" :url="dialogUrl"   @success="operatorDialogSuccess" v-loading="dialogloading">
	</el-dialog>
	<!-- 消息中心，删除消息提示 -->
	<el-dialog title='<%=rb.getString("QueRen")%>' :visible="showConfirmInfo" width="400" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeMsgCenterBtn">
		<span v-if='batchOrSingleDel == "batch"'><%=rb.getString("XiaoXiZhongXinShanChuTiShi")%></span>
		<span v-else><%=rb.getString("XiaoXiZhongXinShanChuTiShiSingle")%></span>
		<div slot="footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="confirmMsgBtn"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeMsgCenterBtn"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-dialog>

	<!-- #69775 磁盘空间不足弹窗提示 -->
    <div class='diskSpaceTooltip' v-show='showDiskSpaceInfo'>
        <p class='head' style='justify-content: space-between; display: flex; line-height: 30px;'>
            <span><%=rb.getString("CiPanGaoJingTiShiBiaoTi")%></span>
            <span @click='diskSpaceIgnoreBtn' style='font-size: 12px; color: #7A7992; cursor: pointer;'><%=rb.getString("Button_HuLue")%></span>
        </p>
        <span class='commonSize14'><%=rb.getString("CiPanGaoJingTiShiNeiRong")%></span>
        <div v-html="diskSpaceInfos" class='commonGeneral12' style='margin: 16px 0 10px; line-height: 12px;' ></div>
        <b @click='closeDiskSpaceInfoBtn'></b>
    </div>
</div>

<%-- 窗口-显示北向接口通信异常告警记录 --%>
<div id="winItfnAlarm" class="easyui-window" title="<%=rb.getString("BeiXiangJieKouGaoJingBiaoTi")%>"
     data-options="modal:true,closed:true,minimizable:false,maximizable:false,collapsible:false,width: 600,height:400">
</div>

<%-- 窗口-显示SAS通信异常告警记录 --%>
<div id="winSasAlarm" class="easyui-window" title="<%=rb.getString("SASgaojing")%>"
     data-options="modal:true,closed:true,minimizable:false,maximizable:false,collapsible:false,width: 1000,height:500">
</div>

<%-- 上传文件时的进度条窗口 --%>
<div id="winUploadPro" title="<%=rb.getString("ShangChuanJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:300,height:80,resizable:false">
    <div id="progressUploadFile" class="easyui-progressbar" style="width:280px;margin-top:2px;margin-left:2px;"></div>
</div>
<!-- 窗口，弹出当前密码策略提示信息 -->
<div id="winPasswdRulePrompt" class="easyui-window" title="<%=rb.getString("MiMaCeLueBiaoTi")%>"
		data-options="modal:true,split:true,closed:true,collapsible:false,minimizable:false,maximizable:false,
			width:400,height:150">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="text-align: center;padding: 15px;">
           	<span><%=rb.getString("MiMaChangDu")%><%=rb.getString("MaoHao")%></span>
           	<span id="passwdMinLengthPrompt" value=""/></span> -
           	<span id="passwdMaxLengthPrompt" value=""/></span>
           	<span><%=rb.getString("ZiFu")%></span>
           	<span><%=rb.getString("DouHao")%></span>

            <span><%=rb.getString("ZhangHuYouXiaoTianShu")%><%=rb.getString("MaoHao")%></span>
            <span id="passwdValidPeriodPrompt" value=""></span>
            <span><%=rb.getString("Tian")%></span>
            <span><%=rb.getString("JuHao")%></span>
		</div>
		<div region="south" data-options="border:false,height:47" style="text-align: center;padding: 10px">
			<a href="#" class="easyui-linkbutton" style="margin-right:15px;" onclick="sureModifyPass()"><%=rb.getString("QueDing")%></a>
			<a href="#" class="easyui-linkbutton" style="" onclick="cancelModifyPass()"><%=rb.getString("QuXiao")%></a>
		</div>
	</div>
</div>

<div id="winDefault" class="easyui-window" title=" "
	data-options="modal:true,split:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:900,height:500,resizable:false">
</div>

<div id="getCellInfoErrorTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoOffTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoOnTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoUpdateTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoInitTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoSyncTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoSyncedTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>

<div id="winopenFull" class="easyui-window" title="全景视图" data-options="modal:true,closed:true,draggable:false,minimizable:false,maximizable:false,collapsible:false"
     fit='true'></div>

<%-- 接入控制导入文件失败后弹出下载框窗口 --%>
<div id="winDowloadFailureFileAccess" class="easyui-window" title="T"
     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:450,height:180,resizable:false,inline:true">
	<div region="center" data-options="border:false" style="padding:20px">
		<span id="failureTextAccess" style="line-height:25px;color:#797979;font-size:14px;"></span>
	</div>
	<div region="south" data-options="border:false" style="height:47px;padding-bottom:20px;">
		<a class="linkbutton" onclick="dowloadFailureFileAccess()" style="float:right;margin-right:20px;"><span><%=rb.getString("XiaZai")%></span></a>
	</div>
</div>
<%--自配置导入文件失败后弹出下载框窗口 --%>
<div id="winDowloadFailureFileConfig" class="easyui-window" title="T"
     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:450,height:180,resizable:false,inline:true">
	<div region="center" data-options="border:false" style="padding:20px">
		<span id="failureTextConfig" style="line-height:25px;color:#797979;font-size:14px;"></span>
	</div>
	<div region="south" data-options="border:false" style="height:47px;padding-bottom:20px;">
		<a class="linkbutton" onclick="dowloadFailureFileConfig()" style="float:right;margin-right:20px;"><span><%=rb.getString("XiaZai")%></span></a>
	</div>
</div>
<%--策略配置导入文件失败后弹出下载框窗口 --%>
<div id="winDowloadFailureFileBatch" class="easyui-window" title="T"
     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:480,height:200,resizable:false,inline:true">
	<div region="center" data-options="border:false" style="padding:20px">
		<span id="failureTextBatch" style="line-height:25px;color:#797979;font-size:14px;"></span>
	</div>
	<div region="south" data-options="border:false" style="height:30px;padding-bottom:20px;">
		<a class="linkbutton" onclick="dowloadFailureFileBatch()" style="float:right;margin-right:20px;"><span><%=rb.getString("XiaZai")%></span></a>
	</div>
</div>
<form id="downloadFailureBatchForm" style="display:none" method="post" action="${ctx}/task/BatchConfiguration/downloadFailureFile.action"></form>
<div class='success_msg'></div>
<div class='error_msg'></div>
<div class='prompt_msg'></div>
<div class='alert_msg'></div>
<div class='licenseTooltip'>
	<p class='head'>
		<span><%=rb.getString("TiShi") %></span>
	</p>
	<p class='message'></p>
	<b onclick='closeLicenseTip()'></b>
</div>
<div class='maskWindow' style='opacity:0.6;display:none;z-index:1100;position:absolute;left:0;right:0;top:0;bottom:0;background:#CDE0E8'></div>
<script type="text/javascript">
var headerType = "${Types}";  //是否由cloud接入
var isSueperAdmin_var = "${isSuperAdmin}";
var isDistributor = "${isDistributor}";
var collectInterval = null;
var gsmCollectInterval = null;
var cpeCollectInterval = null;
var progressInterval = null;
var gnbProgressInterval = null;
var gsmProgressInterval = null;
var navTabsEnable = true;
var supportTopoSite = '${supportTopoSite}' == 1;
var supportGSM = '${gsmEnable}' == 'true';
var isSupportSSO = '${isSupportSSO}' == 'true';
var isFenceEnable = '${fenceEnable}' == 'true';

//定时器-磁盘告警提示
var diskSpaceAlarmInterval;
//定时器-获取消息中心信息推送
var updateMessageCenterTimer;

var uiCustomOld = {
		"ui_color":"#FF4614",
		"ui_login_background":"./images/login/login_bg.png",
		"ui_menu_logo_up":"./images/login/nav_logo_collapse.png",
		"ui_menu_logo_down":"./images/login/logo_big.png",
		"ui_restore":"true",
		"ui_omc_name":""
};
var uiCustom = {
		"ui_color":"#FF4614",
		"ui_login_background":"./images/login/login_bg.png",
		"ui_menu_logo_up":"./images/login/nav_logo_collapse.png",
		"ui_menu_logo_down":"./images/login/logo_big.png",
		"ui_restore":"false",
		"ui_omc_name":"OMC"
}
var colorUIRgb = changeColor(uiCustom.ui_color);
try{// 强制更新cookie
	if ( '${isReload}' != 'true'){
		document.cookie = 'smallcell=${cookieSessionId};path=/;domain=' + window.location.hostname;
	}
}catch(e){
	console.log(e)
}

$.fn.form.defaults.queryParams.token =  '${token}';

$.ajaxSetup({
	headers: {
		token: '${token}'
	},
	beforeSend: function(){
		var paramstr = arguments[1].data,bool=true;
		if(paramstr){
			var paramArr = paramstr.split('&');
			paramArr.map(function(item){
				var codes = item.split('=');
				if(!validXSS(codes[1])) bool = false;
			});
		}
		if(!bool) {
			try{
				$('.messager-window:contains(tips)').panel('destroy');
			}catch(e){}
			toast(XSSValidInfo,'#omc_app_ctn');
			setTimeout(function(){
				$('.datagrid-mask,.datagrid-mask-msg').hide();
			},1500);
		}
		return bool;
	}
});

axios.defaults.headers.common['token'] = '${token}';

/* 通过ajax抓取数据 */
var accessJson = [],writableMap = {};

$.ajax({
	type:'post',
	url:'${ctx}/sys/login/getFeatureCodesInfo.action',
	dataType: 'json',
	async: false,
	success: function(data){
		if(data){
			accessJson = data;

			/* 初始化code 和 writable的键值对集 */
			try{
				accessJson.map(function(item){
					writableMap[item.key] = item.writable;
				});
			}catch(e){}
		}
	},
	error: function(){
		/* 初始化code 和 writable的键值对集 */
		try{
			accessJson.map(function(item){
				writableMap[item.key] = item.writable;
			});
		}catch(e){}
	}
});

if(sessionStorage.locked){}else $('body').removeClass('loading');

if(isMobile()) {
	checkMobileOnWindowResize();
}

// resize事件防抖处理 - 优化窗口大小变化时的性能
var resizeDebounceTimer = null;
var debouncedResize = function() {
	if (resizeDebounceTimer) {
		clearTimeout(resizeDebounceTimer);
	}
	resizeDebounceTimer = setTimeout(function() {
		checkMobileOnWindowResize();
		// 触发自定义事件，供子页面监听
		window.dispatchEvent(new CustomEvent('debouncedResize'));
	}, 150); // 150ms防抖延迟
};

window.removeEventListener('resize', debouncedResize);
window.addEventListener('resize', debouncedResize);

var sysMain = new Vue({
	el:'#sysMain',
	data(){
		let navList = [],
            validatorDiskSpaceItem = (rule,value,callback) => {
                if(value === '' || value === null || value === undefined){
                    callback(new Error('<%=rb.getString("QingXuanZe")%>'))
                }else{
                    callback();
                }
            };
		return{
			navTabsEnable: navTabsEnable === true,
			navList: navList,

			oldSubMenuLabel: '',
			oldActive: '',
			netOldActive: '',
			headType: sessionStorage.getItem('neType') || 'enb',
			headBtnData: [],
			menuDefaultActive: '',
			menus: [],

			noItfnAlarmBtn:'${noItfnAlarmBtn}',
			headerClass:'sysHeader',
			type:headerType,
			headPath:[''],
			headerHeight:'36px',
			menuData:[],
			isCloud:isCloud,
			isSuperAdmin:isSueperAdmin_var,
			isSuper_user:is_super_user,
			menuContStyle:{
				width:'54px'
			},
			popoverWidth:{
				width:'220px'
			},
			isCollapse:true,
			//muneToggleIcon:'muneOpenIcon',
			logoCollapseClass:'el-icon-logo-omc',
			mainPageUrl:'',
			colorHighLight:'colorHighLight',
			colorHui:'colorHui',
			showVer:false,
			defaultActive: '1000',
			menuChildId:'',
			menuChildData:'',
			menufristId:1,
			showBrowserTip:false,
			showVersion:true,
			versionSpan:'versionCollapseStyle',
			showChangeOperator:false,
			operatorDialogWidth:'90%',
			dialogUrl:'',
			dialogloading:false,
			migrateRole:'',

			msgUrl: '${ctx}/msgCenter/queryMsgCenterPageList.action',

			messsgeCenterShowOrHide: false,
			showConfirmInfo: false,
			circleTipShowOrHide: false,
			inprogressTipShowOrHide: false,
			msgTable_params: {
				timeZone : timeZone
			},
			curMsgData: [],
			batchOrSingleDel: '',
			curDelId: '',

			unreadFlag:isUnread,

			showDiskSpaceInfo: false,
			diskSpaceInfos: '', //告警信息
		}
	},
	computed:{
		muneToggleIcon:function(){
			return this.isCollapse ? 'el-icon el-icon-common-right' : 'el-icon el-icon-common-left'
		},
		fontSizeColor:function(){

		},
		netRoles() {
			var vm = this,
				code = vm.defaultActive,
				map = {
					'1': ['enb','cpe','gnb'],
					'1000': ['enb','cpe','gnb'],
					'2000': ['enb','cpe','gnb'],
					'3000': ['enb','cpe','gnb','egw','ups'],
					'4000': ['enb','cpe','gnb','egw','ups'],
					'5000': ['enb','cpe','gnb','egw','ups'],
					'6000': ['enb','cpe','gnb','egw'],
					'7000': [],
					'8000': [],
					'9000': ['enb','gnb','egw'],
					'10000': [],
					'11000': [],
				},
				roles = map[code] || [];

			return roles;
		}
	},
	methods:{
		/*
		阐述问题： 为解决 #107225 问题单
		1、网元选中 eNB;  
		2、点击左侧菜单 Inventory； 
		3、点击二级菜单 Data Model; 
		4、三级页签显示 eNB - Data Model;  
		5、此时关闭 eNB - Data Model 页签； 
		6、去切换不支持 Data Model 的网元 WCG； 此时三级页签eNB - Data Model 自动显示，点击页面内容是空白； 
		7、实际上已关闭三级的页签，在切换网元后，都不应该显示
		8、是否存在缓存问题？ 
		9、切换网元，当前还存在的页签是不应该被清理的；  只需把手动关闭的页签不要再展示
		-------------------------------------------------------------------------------------------------------
		10、点击 Inventory；2、点击二级菜单 Data Model 3、手动关闭 Data Model 页签； 4、切换网元到 gNB, eNG-Data Model 页签却还显示
		解决方案：
		1、在 onTabRemoved 事件中，记录被删除的页签菜单名称到 oldSubMenuLabel；
		2、在 loadSubMenus 方法中，优先通过 oldSubMenuLabel 查找对应菜单项，如果找到则打开该菜单；
		3、如果没有找到，则继续通过 sessionStorage.submenuid 查找对应菜单项，如果找到则打开该菜单；
		4、如果两者都没有找到，则打开第一个菜单项。
		5、在 onTabRemoved 事件中，修复只清除与被删除页签匹配的 oldSubMenuLabel 和 sessionStorage.submenuid，避免影响其他菜单的状态。
		*/
		onTabRemoved(removedTab) {
			let vm = this;
			if(removedTab && removedTab.title) {
				// 提取被删除页签的菜单名称（去除网元前缀）
				let menuName = removedTab.title;
				if(menuName.indexOf(' - ') > -1) {
					menuName = menuName.split(' - ')[1];
				}
								
				// 修复：只清除与被删除页签匹配的 oldSubMenuLabel
				// 如果用户关闭的是当前记录的菜单，则清空；否则保留其他菜单的状态
				if(vm.oldSubMenuLabel === menuName) {
					vm.oldSubMenuLabel = '';
				}
				
				// 修复：只清除与被删除页签匹配的 sessionStorage
				// 如果用户关闭的是当前缓存的菜单，则清空；否则保留
				if(removedTab.menuId) {
					let storedMenuId = sessionStorage.getItem('submenuid');
					if(storedMenuId == removedTab.menuId) {
						sessionStorage.removeItem('submenuid');
					} 
				}
			}
		},
		headTypeChange(type) {
			let vm = this,
				menuId = vm.defaultActive;

			sessionStorage.setItem('neType', type);

			if(menuId == '1000' || menuId == '8000') return;

			vm.loadSubMenus(menuId, type);
		},
		loadSubMenus(menuId, netType, fn) {
			let vm = this,
				type = netType.toLowerCase();

			if(['7000','11000'].includes(menuId+'')) {
				type = '';
			}
			if(['egw','ups'].includes(netType) && ['7000'].includes(menuId+'')) {
				type = netType.toLowerCase();
			}

			$.post('${ctx}/system/sysGlb/getLastMenuTree.action',{menuId: menuId, type: type},function(data){
				vm.menus = data || [];

				if(vm.menus.length == 0) {
					vm.$message.error('<%=rb.getString("WangYuanBuZhiChi")%>');
					vm.defaultActive = vm.oldActive;
					vm.headType = vm.netOldActive;

					var selectedDom = document.querySelectorAll('#menuAnimate .is-active,#menuAnimate .selectSty');
					if(selectedDom.length !==0){
						Array.from(selectedDom).map(function(item){
							item.classList.remove('selectSty');
							item.classList.remove('is-active');
						});
					}

					var reselectedDom = document.querySelector('#menu_' + vm.defaultActive);
					if(reselectedDom){
						reselectedDom.classList.add('selectSty');
						reselectedDom.classList.add('is-active');
					}

					return;
				}

				vm.oldActive = menuId;
				vm.netOldActive = netType;

				var matchItem = vm.menus.filter(function(menu){
					return menu.menu_name == vm.oldSubMenuLabel;
				});

				if(matchItem.length) {
					vm.menuSelect(matchItem[0].menu_url, fn, netType);
				}else {
					vm.menus.map(function(menu, idx){
						if(idx == 0) {
							if(menu.children && menu.children.length) {
								vm.menuSelect(menu.children[0].menu_url, fn, netType);
							}else {
								var subId = sessionStorage.getItem('submenuid');
									subItem = vm.menus.filter(function(menu){
										return menu.id == subId;
									});

								if(subItem.length) {
									vm.menuSelect(subItem[0].menu_url, fn, netType);
								}else {
									vm.menuSelect(menu.menu_url, fn, netType);
								}
							}
						}
					});
				}

				if(menuId) {
					vm.defaultActive = menuId;

					var sledDom = document.querySelectorAll('#menuAnimate .is-active,#menuAnimate .selectSty');
					if(sledDom.length !==0){
						Array.from(sledDom).map(function(item){
							item.classList.remove('selectSty');
							item.classList.remove('is-active');
						});
					}

					var rsledDom = document.querySelector('#menu_' + vm.defaultActive);
					if(rsledDom){
						rsledDom.classList.add('selectSty');
						rsledDom.classList.add('is-active');
					}
				}
			},'json');
		},
		menuSelect(url, fn, netType) {
			let vm = this,
				netorkType = netType || vm.headType;

			var matchItem = vm.menus.filter(function(menu){
					return menu.menu_url == url;
				});

			vm.oldSubMenuLabel = matchItem[0].menu_name;
			vm.menuDefaultActive = '';

			if(vm.navTabsEnable) {
				let netItem  = vm.headBtnData.filter(function(item){
						return item.code == netorkType;
					}),
					netLabel = netItem.length ? netItem[0].label + ' - ' : '';

				if(['1000', '8000','700001','200001','200002','200003'].includes(matchItem[0].id+'') || ['11000'].includes(matchItem[0].pid+'')) {
					netLabel = '';
				}

				vm.$refs.nav.addTab({
					netType: vm.headType,
					url: matchItem[0].menu_url,
					title: netLabel + matchItem[0].menu_name,
					id: matchItem[0].id+'',
					netType: netorkType,
					callback: ()=> {
						vm.menuDefaultActive = url;

						if(fn && typeof fn == 'function') {
							try{
								fn();
							}catch(e){}
						}

						sessionStorage.setItem('submenuid', matchItem[0].id);
					}
				});
			}else {
				$('#mainpage').addClass('loading');
				$('#mainpage').load( '${ctx}'+ url,function(){
					$.parser.parse(this);
					$('#mainpage').removeClass('loading');
					vm.menuDefaultActive = url;

					if(fn && typeof fn == 'function') {
						try{
							fn();
						}catch(e){}
					}

					sessionStorage.setItem('submenuid', matchItem[0].id);
				});
			}
			// 隐藏popover用
			// document.body.click();
			Array.from(document.querySelectorAll('.el-popover.el-popper')).map(function(node){
				try{
					node['__vue__'].$parent && node['__vue__'].$parent.doClose();
				}catch(e){}
			});
		},
		showUnreadAlarm() {
            var params = {
                    alarm_severity: '',
                    unread: '1',
                    search_text: ''
                };
            try{
                eventAllBus.$emit("gomenupage","8000","",'8000',params,function() {
                    alarmViewVue.activeName = 'alarmView';
                    alarmViewVue.resetQueryParams(params);
                });
                if(alarmViewVue) {
                    alarmViewVue.activeName = 'alarmView';
                    alarmViewVue.resetQueryParams(params);
                }
            }catch(e){}
		},
		showCurrAliveAlarm(alarm_severity) {
			var title = "";
			if (alarm_severity == '31001') {
				title = "Critical Alarms";
			} else if (alarm_severity == '31002') {
				title = "Major Alarms";
			} else if (alarm_severity == '31003') {
				title = "Minor Alarms";
			} else if (alarm_severity == '31004') {
				title = "Warning Alarms";
			}
			try{
                var params = {
                    alarm_severity: alarm_severity,
                    unread: '',
                    search_text: ''
                };
                eventAllBus.$emit("gomenupage","8000","",'8000',params,function() {
                    alarmViewVue.activeName = 'alarmView';
                    alarmViewVue.resetQueryParams(params);
                });
                if(alarmViewVue) {
                    alarmViewVue.activeName = 'alarmView';
                    alarmViewVue.resetQueryParams(params);
                }
            }catch(e){}

		},

		commandHandle(command){
			switch (command){
				case "changePassWord":
					passwdUpd();
					break;
				case "changeOperator":
					this.changeOperator();
					break;
				case "lockScreen":
					screenLock();
					break
				case "logout" :
					if(isSupportSSO == true && isCloud != 'true') {
						this.logoutSSO();
					}else {
						logout('${ctx}');
					}
					break
			}
		},
	logoutSSO(ctx) {
			// 使用页面跳转而不是 AJAX：SAML SP 发起登出需要浏览器完整重定向，
			// 这样才能接收 IdP 返回的 LogoutResponse 并继续后续流程。
			var registrationId = 'keycloak';
			var logoutUrl = '${ctx}/saml2/logout/' + registrationId;
			// 避免重复点击
			if(window.__slo_in_progress){return;}
			window.__slo_in_progress = true;
			// 使用隐藏表单 POST 提交代替 AJAX，保证浏览器执行完整导航并能跟随 IdP 重定向
			try{
				var form = document.createElement('form');
				form.method = 'POST';
				form.action = logoutUrl;
				form.style.display = 'none';
				document.body.appendChild(form);
				form.submit();
			} catch(e){
				// 兜底：若表单提交失败，再使用 href 跳转
				window.location.href = logoutUrl;
			}
		},
		init(){
			var vm = this;


			//查询UI定制信息 ，如果 使用默认配置，则将old参数赋值到当前ui 否则，将查询结果赋值
			$.post("${ctx}/ui/customization/getCustomizationInfo.action",{},function(data){
		 		if(data["ui_restore"] == "true"){
		 			Object.assign(uiCustom,uiCustomOld);
		 			colorUIRgb = changeColor(uiCustom.ui_color);
                    uiCustomOld.ui_omc_name = document.title;
					document.documentElement.style.setProperty("--main-color-rgba1" , colorUIRgb);
				  	document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustom.ui_login_background +')');
					document.documentElement.style.setProperty("--main-color" ,uiCustom.ui_color);
					document.documentElement.style.setProperty("--logo-small" , 'url('+ uiCustom.ui_menu_logo_up +')');
					document.documentElement.style.setProperty("--logo-big" ,'url('+ uiCustom.ui_menu_logo_down +')');
					document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustom.ui_menu_logo_down +')');
		 	    }else {
		 		  	Object.assign(uiCustom,data);
		 		  	colorUIRgb = uiCustom.ui_color? changeColor(uiCustom.ui_color): '255,70,20';

                    document.title = uiCustom.ui_omc_name;
					document.documentElement.style.setProperty("--main-color-rgba1" , colorUIRgb);
				  	document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustom.ui_login_background +')');
					document.documentElement.style.setProperty("--main-color" ,uiCustom.ui_color);
					document.documentElement.style.setProperty("--logo-small" , 'url('+ uiCustom.ui_menu_logo_up +')');
					document.documentElement.style.setProperty("--logo-big" ,'url('+ uiCustom.ui_menu_logo_down +')');
					document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustom.ui_menu_logo_down +')');
		 	    }
				updateFavicon(uiCustom.ui_color);
		    },"json")

			if(vm.navTabsEnable) {
				if("${menu_id}") {

				}else {
					vm.$refs.nav.addTab({
						netType: vm.headType,
						url: '${ctx}/sys/menu/goHomeMenu.action',
						title: 'Dashboard',
						id: '1000',
						callback: ()=> {
							vm.defaultActive = 1000;
						}
					});
				}
			}else {
				$('#mainpage').load( '${ctx}/sys/menu/goHomeMenu.action',function(){

				});
			}

	        axios.post("${ctx}/system/sysuser/queryLoginPrompt.action").then(function(response){
				var data = response.data;
				if (data["TITLE"] && "${ispwdtitle}"!="true") {
        			vm.$message({
        				message:data["MSG"],
        				type:'warning',
        				showClose:true,
        				dangerouslyUseHTMLString:true,
        				duration:0
        			})
            	}
			});

	        axios.post("${ctx}/system/sysGlb/getSupportCodes.action").then(function(response){
	        	let list = response.data||[];

	        	vm.headBtnData = list;

	        	vm.headBtnData.map(function(item, idx){
	        		if(idx == 0) {
	        			vm.headType = item.code || sessionStorage.getItem('neType');
						vm.netOldActive = vm.headType;
						sessionStorage.setItem('neType', vm.headType);
	        		}
	        	});
			});

			// "${ctx}/system/sysGlb/getMenuTree.action"
			axios.post("${ctx}/system/sysGlb/getNewMenuTree.action",stringify({
				top_id:'0',
			})).then(function(response){
				vm.menuData = response.data;
				vm.$nextTick(()=>{

					Array.from(document.querySelectorAll('.el-submenu__title >.el-icon-arrow-down')).map((item)=>{
						// item.classList.remove('el-icon-arrow-down');
						// item.classList.add('el-icon-arrow-right');
					})
					var style = document.createElement('style');
					style.innerHTML = `
						.selectSty{
							padding-left:20px!important;
						}
					`;
					document.body.appendChild(style);
					document.querySelector('#menu_'+1000).classList.add('selectSty');

					if("${menu_id}") {
						eventAllBus.$emit("gomenupage","${menu_id}","","${menu_id}",false,function() {

						});
					}
				})
				/* vm.menuData = response.data.splice(2,1) */
			});

			axios.post("${ctx}/system/sysuser/queryLoginPrompt.action").then(function(response){
				var data = response.data;
				if (data["TITLE"] && "${ispwdtitle}"!="true") {
        			vm.$message({
        				message:data["MSG"],
        				type:'warning',
        				showClose:true,
        				dangerouslyUseHTMLString:true,
        				duration:0
        			})
            	}
			});
			axios.post("${ctx}/migrate/getMigrateRole.action").then(function(response){
				var data = response.data;
				vm.migrateRole = data.migrateRole;
			});

            //磁盘告警信息查询
            if(is_super_user == 'true'){
                vm.getDiskSpaceInfo();
            }

			//是否有新消息
            vm.getMsgCenterNews();

			if ( vm.type == 'true'){
				vm.headerClass = 'sysHeader white'
			}
			if("${Types}" == "false"){//local版本
				vm.showVersion = true;
				var explorer = window.navigator.userAgent;
				var explorFlag = false;
				if(explorer.indexOf("Firefox") > 0){//火狐浏览器
					explorFlag = true;
				}else if(explorer.indexOf("Edg") > 0){//Edge浏览器
					explorFlag = true;
				}else if(explorer.indexOf("Mac") > 0){//Safari浏览器
					explorFlag = true;
				}else if(vm.isChrome()){//Google浏览器
					explorFlag = true;
					// 完全使用chromiun内核，且不做任务差异处理的
					try{
						var mimeTypes = Array.from(window.navigator.mimeTypes).map(function(item){ return item.type;}),
							excludes = ['application/360softmgrplugin','application/x-ppapi-widevine-cdm'];

						excludes.map(function(item){
							if(mimeTypes.includes(item)) explorFlag = false;
						})
					}catch(e){}
				}else{
					explorFlag = false;
				}
				if(explorFlag){//此浏览器是支持的浏览器
					vm.showBrowserTip = false;
				}else{
					vm.showBrowserTip = true;
				}
			}else{//cloudcore版本
				vm.showVersion = false;
				vm.showBrowserTip = false;
			}
			let userCode = sessionStorage.getItem('userCode');
			if(userCode != user_code){
				sessionStorage.setItem('keyData','');
			}
		},
		menuCollapseClick(){
			var vm = this;
			vm.messsgeCenterShowOrHide = false; // 关闭消息中心
			if ( vm.isCollapse == false){   //菜单当前为打开状态
				vm.isCollapse = true;
				/* vm.muneToggleIcon = 'muneCloseIcon';  */
				vm.logoCollapseClass = 'el-icon-logo-omc';
				vm.menuContStyle = {
						width:'54px'
				};
				vm.showVer = false;
				vm.versionSpan = "versionCollapseStyle";
				setTimeout(()=>{
					Vue.nextTick(()=>{
						if(document.querySelectorAll('.selectSty').length !==0){
							document.querySelectorAll('.selectSty')[0].classList.remove('selectSty')
						}
						var style = document.createElement('style');
						style.innerHTML = `
							.selectSty{
								padding-left:20px!important;
							}
							.selectSty .el-submenu__title{
								padding-left:0px!important;
							}
						`;
						document.body.appendChild(style);
						setTimeout(()=>{
							document.querySelector('#menu_'+vm.menufristId).classList.add('selectSty')
						},500)
						if(vm.menuChildId){
							var ids = vm.menuChildId+'';
							vm.$refs.menuList.activeIndex = vm.menuChildId;
							vm.$refs.menuList.defaultActive = ids;
						}
					})
					window.dispatchEvent(new Event('resize'));
				},500)
				window.dispatchEvent(new Event('resize'));
			}else{
				vm.isCollapse = false;
				/* vm.muneToggleIcon = 'muneOpenIcon';  */
				vm.logoCollapseClass = 'el-icon-logo-baiomc';
				vm.menuContStyle = {
						width:'200px'
				};

				vm.showVer = true;
				vm.versionSpan = "";
				setTimeout(()=>{
					Vue.nextTick(()=>{
						Array.from(document.querySelectorAll('.el-submenu__title >.el-icon-arrow-down')).map((item)=>{
							// item.classList.remove('el-icon-arrow-down');
						})
						if(vm.menufristId){
								var ids = vm.menufristId+'';
								if(document.querySelectorAll('.selectSty').length !==0){
									document.querySelectorAll('.selectSty')[0].classList.remove('selectSty');

								}
								var style = document.createElement('style');
								style.innerHTML = `
									.selectSty{
										padding-left:20px!important;
									}
									.selectSty .el-submenu__title{
										padding-left:0px!important;
									}
								`;
								document.body.appendChild(style);
								setTimeout(()=>{
									document.querySelector('#menu_'+vm.menufristId).classList.add('selectSty')
								},50)
						}
					})
					window.dispatchEvent(new Event('resize'));
				},600)
				window.dispatchEvent(new Event('resize'));
			}
			setTimeout(()=>{
				eventBus.$emit('dashboard-resize');
			},500)
		},
		menuClick(index,indexPath,e,params,fn){
			var vm = this,
				queryParams = {menu_id: index};
			//点击菜单 e是菜单dom 对象 点击其他地方跳转 e是对应的id 做相应的处理
			let url = ""
			let path = ""
			vm.messsgeCenterShowOrHide = false; // 关闭消息中心
			if(vm.isCollapse == false){ // 导航菜单展开状态
				var menuSelectOne = document.querySelectorAll('#menu_'+index)
				if(menuSelectOne.length !== 0){
					var menuSelectPop = $('#menu_'+index).attr('poper-class');
					if(menuSelectPop == '1'){
						vm.addSelectEvent(index);
						vm.menufristId = index+'';
						vm.menuChildId = '';
						vm.$refs.menuList.activeIndex = index;
						vm.$refs.menuList.defaultActive = index+'';
						if(typeof e == "string"){
							index = e;
							var _id = "menu_"+e
							e = $("#"+_id)
							url = $(e).attr('url');
							path = $(e).attr('path');
						}else{
							url = $(e)[0].$attrs.url;
							path = $(e)[0].$attrs.path;
						}
					}else{
						var vm = this;
						var e = "#menu_"+index;
						var fatherId = $(e).attr('fatherId');
						url = $(e).attr('url');
						path = $(e).attr('path');
						vm.menufristId = fatherId;
						vm.menuChildId = index;
						vm.$refs.menuList.activeIndex = index;
						vm.$refs.menuList.defaultActive = index+'';
						vm.addSelectEvent(fatherId,0);
					}


				}

			}

			if(vm.isCollapse == true){ // 导航菜单收起状态
				var menuSelectVal = document.querySelectorAll('#menu_'+index)
				vm.menuChildId = index;
				vm.menufristId = $('#menu_'+index).attr('fatherId');
				vm.addSelectEvent(vm.menufristId);
				vm.$refs.menuList.activeIndex = index;
				vm.$refs.menuList.defaultActive = index+'';
				if(typeof e == "string"){
					index = e;
					var _id = "menu_"+e
					e = $("#"+_id)
					url = $(e).attr('url');
					path = $(e).attr('path');

				}else{
					url = $(e)[0].$attrs.url;
					path = $(e)[0].$attrs.path;
				}
			}

			//显示当前的菜单路径
			vm.headPath = path.split(">");   //.join("/");
			//判断跳转是否有参数
			let paramsStr = "";
			if(params){
				Object.assign(queryParams, params);
			}
			for (var key in queryParams) {
				paramsStr += "&" + key + "=" + queryParams[key];
			}
			var urlArr = url.split('?');
			if(urlArr[1]) {
				url += ("&omcVersion="+ omcVersion + paramsStr)
			}else{
				url += ("?omcVersion="+ omcVersion + paramsStr)
			}

			if ( index == '4002'){
				openDefaultWindow('${ctx}'+ url,{
		    		title: '<%=rb.getString("GuanYu")%>',
		    		width:400,height:180
		    	});
			}else{
				vm.menus = [];

				if(url.indexOf('.action')>=0 || url.indexOf('/')>=0) {
					var selectedDom = document.querySelectorAll('#menuAnimate .is-active,#menuAnimate .selectSty');
					if(selectedDom.length !==0){
						Array.from(selectedDom).map(function(item){
							item.classList.remove('selectSty');
							item.classList.remove('is-active');
						});
					}

					if(vm.navTabsEnable) {
						let netItem  = vm.headBtnData.filter(function(item){
							return item.code == vm.headType;
						}),
						netLabel = netItem.length ? netItem[0].label + ' - ' : '';

						if(['8000','1000','200001','200002','200003'].includes(index+'')) {
							netLabel = '';
						}

						vm.$refs.nav.addTab({
							netType: vm.headType,
							url: '${ctx}'+ url,
							title: netLabel + vm.headPath[vm.headPath.length-1],
							id: index+'',
							callback: ()=> {
								if(index) {
									vm.defaultActive = index;
								}

								if(fn && typeof fn == 'function') {
									try{
										fn();
									}catch(e){}
								}
							}
						});

						if(index) {
							vm.defaultActive = index;
						}
					}else {
						$('#mainpage').load( '${ctx}'+ url,function(){
							$.parser.parse(this);
							if(index) {
								vm.defaultActive = index;
							}
							if(fn && typeof fn == 'function') {
								try{
									fn()
								}catch(e){}
							}
						});
					}
				}else {
					vm.loadSubMenus(index, vm.headType, fn);
				}

				$.post('${ctx}/system/sysGlb/setSessionMenuId.action?menu_id='+index);
				// 隐藏popover用
				//document.body.click();
				Array.from(document.querySelectorAll('.el-popover.el-popper')).map(function(node){
					try{
						node['__vue__'].$parent && node['__vue__'].$parent.doClose();
					}catch(e){}
				})
			}
		},
		isChrome(){
			var agent = window.navigator.userAgent,
				reg = /^[\s\S]+Chrome\/[0-9\. ]+(\sMobile\s)?Safari\/[0-9\.]+$/;

			return reg.test(agent);
		},
		// 导航打开事件
		menuOpen(index){
			var vm = this;
			vm.openMenuId = index+'';
			vm.$refs.menuList.close(vm.openMenuId);
		},
		// 导航关闭事件
		menuClose(index){
			var vm = this;
		},
		// 添加选中样式
		addSelectEvent(id,time){
			var selectedDom = document.querySelectorAll('#menuAnimate .selectSty');
			if(selectedDom.length !==0){
				Array.from(selectedDom).map(function(item){
					item.classList.remove('selectSty');
				});
			}

			var style = document.createElement('style');
			style.innerHTML = `
				.selectSty{
					padding-left:20px!important;
				}
				.selectSty .el-submenu__title{
					padding-left:0px!important;
				}
			`;
			document.body.appendChild(style);
			if(time == 0){
				document.querySelector('#menu_'+id).classList.add('selectSty')
			}else{
				setTimeout(()=>{
					document.querySelector('#menu_'+id).classList.add('selectSty')
				},20)
			}
		},
		// 浮窗打开事件
		popoverShow(val){
			var vm = this,twoMenuNum,threeMenuNum = 0,sumNum;
			var showMenuData = vm.menuData.find((item)=>{
				return item.id == val
			})
			twoMenuNum = showMenuData.children.length;

			showMenuData.children.map((item)=>{
				if(item.children){
					threeMenuNum +=item.children.length
				}
			})
			sumNum = (twoMenuNum*40) + (threeMenuNum*30);
			if(sumNum > 430){
				vm.popoverWidth = {
						width:'440px'
				};
			}else{
				vm.popoverWidth = {
						width:'220px'
				};
			}
			if(document.querySelectorAll('.popoverClick').length !==0){
				document.querySelectorAll('.popoverClick')[0].classList.remove('popoverClick');
			}

			setTimeout(()=>{
				document.querySelector('#menu_'+val).classList.add('popoverClick')
			},20)
		},
		// 浮窗关闭事件
		popoverHide(val){
			var vm = this;
			if(document.querySelectorAll('.popoverClick').length !==0){
				document.querySelector('#menu_'+val).classList.remove('popoverClick');
			}
			if(vm.isCollapse == true){
				var menufristID =  vm.menufristId;
				vm.$refs.menuList.defaultActive = menufristID;
				vm.$refs.menuList.activeIndex = vm.menufristId;
				vm.addSelectEvent(vm.menufristId)
			}
		},
		menuChildrenClick(item,val){
			var vm = this;

			var selectedDom = document.querySelectorAll('#menuAnimate .selectSty');
			if(selectedDom.length !==0){
				Array.from(selectedDom).map(function(item){
					item.classList.remove('selectSty');
				});
			}

			vm.menufristId = val.id;
			vm.menuChildData = item;
			vm.menuChildId = item.id;
			var ids = item.id+'';
			var menufristID =  val.id+'';
			vm.$refs.menuList.defaultActive = menufristID;
			vm.$refs.menuList.activeIndex = val.id;
			eventAllBus.$emit("gomenupage",ids,"",ids,false);
			vm.$refs.menuPopover.map((item)=>{
				item.doClose();
			});
		},
		// 切换运营商
		changeOperator(){
			var vm = this,
				str = Math.random().toString();
			vm.dialogloading = true;
			vm.showChangeOperator = true;
			vm.dialogUrl = '${ctx}/system/operator/toChangeOperatorPage.action?randomCode='+str;
		},
		// 运营商弹窗加载成功回调
		operatorDialogSuccess(){
			var vm = this;
			setTimeout((e)=>{
				vm.dialogloading = false;
			},1000)

		},
		// 关闭运营商弹窗
		closeOperatorDialog(){
			var vm = this;
			vm.showChangeOperator = false;
		},
		// 设备迁移
		deviceMigrationClick(){
			var vm = this,
				str = Math.random().toString();
			vm.dialogloading = true;
			vm.showChangeOperator = true;
			vm.dialogUrl = '${ctx}/migrate/toDeviceMigration.action?randomCode='+str;
		},

		//消息中心- 有新消息显示小红点； 有进行中的任务-显示loading动态图
		getMsgCenterNews(){
			var vm = this;

            axios.post("${ctx}/msgCenter/getMsgCenterLatestNews.action",stringify({ })).then(function(response){
                var data = response.data;

                if(data.newMsg == true || data.newMsg == 'true'){
                    //当前消息中心弹窗打开状态时，如果新增未读消息，也将红点去除；
                    /*if(vm.messsgeCenterShowOrHide == true){
                        vm.circleTipShowOrHide = false;
                    }else{
                        vm.circleTipShowOrHide = true;
                    }*/
                    vm.circleTipShowOrHide = true;
                }else{
                    vm.circleTipShowOrHide = false;
                }
                if(data.inProgressMsg == true || data.inProgressMsg == 'true'){
                    vm.inprogressTipShowOrHide = true;
                }else{
                    vm.inprogressTipShowOrHide = false;
                }
            });
		},

		//消息中心-表格加载成功
		msgTableLoadSuccess(data){
			var vm = this;
			vm.curMsgData = data.rows;
		},
		//消息中心-批量删除
		batchDeleteBtnClick(){
			var vm = this;
			vm.batchOrSingleDel = 'batch';
			vm.showConfirmInfo = true;
			vm.messsgeCenterShowOrHide = true;
		},
		//删除信息
		confirmMsgBtn(){
			var vm = this;
			if(vm.batchOrSingleDel == 'batch'){
				axios.post("${ctx}/msgCenter/delMsgCenter.action",stringify({
					delType : 'batch'
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type: 'success'
						});
						vm.$refs.msgTable.refresh();
						vm.showConfirmInfo = false;
					}else{
						vm.$message({
							message: data["message"],
							type: 'error'
						});
					}
				});
			}else{
				axios.post("${ctx}/msgCenter/delMsgCenter.action",stringify({
					id : vm.curDelId
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type: 'success'
						});
						vm.$refs.msgTable.refresh();
						vm.showConfirmInfo = false;
					}else{
						vm.$message({
							message: data["message"],
							type: 'error'
						});
					}
				});
			}
		},
		closeMsgCenterBtn(){
			var vm = this;
			vm.showConfirmInfo = false;
			vm.messsgeCenterShowOrHide = true;
		},
		//消息中心-单个删除
		singleDeleteBtnClick(id){
			var vm = this;
			vm.curDelId = id;
			vm.batchOrSingleDel = 'single';
			vm.showConfirmInfo = true;
			vm.messsgeCenterShowOrHide = true;
		},
		//消息中心-弹窗关闭
		closeMessageCenter(){
			var vm = this;
			vm.messsgeCenterShowOrHide = false;
		},

		//获取磁盘信息
		getDiskSpaceInfo(){
		    var vm = this;

            axios.post('${ctx}/fault/prompt/getLatestAlarmPrompt.action').then(function(response){
                var data = response.data;

                if(data){
                    //磁盘告警提示显示条件
                    if(data.ids == '' || data.ids == null || data.ids == undefined){
                        vm.showDiskSpaceInfo = false;
                    }else{
                        vm.showDiskSpaceInfo = true;
                    }
                    if(data.detail == '' || data.detail == null || data.detail == undefined){
                        vm.diskSpaceInfos = '';
                    }else{
                        vm.diskSpaceInfos = data.detail;
                    }
                }else{
                    vm.showDiskSpaceInfo = false;
                }
            }).catch(function(error){})
		},

        //磁盘空间不足，点击忽略按钮-将定时器关闭，不再继续弹出提示；
        diskSpaceIgnoreBtn(){
            var vm = this;
            if (diskSpaceAlarmInterval) {
               clearInterval(diskSpaceAlarmInterval);
            }
            vm.showDiskSpaceInfo = false;
        },

		//磁盘空间不足-弹窗关闭
		closeDiskSpaceInfoBtn(){
		    var vm = this;
            vm.showDiskSpaceInfo = false;
		},
	},
	mounted(){
		this.init();
		eventAllBus.$off("close_OperatorDialog").$on("close_OperatorDialog",this.closeOperatorDialog);
		eventAllBus.$off("gomenupage").$on("gomenupage",this.menuClick);

        /* 菜单隐藏处理 */
        $('#omc_app_ctn').mousedown(function(){
            try{
                var target = event.target, list = Array.from(target.classList),
                    plist = Array.from(target.parentNode.classList);
                if(!(list.includes('filter-menu') || list.includes('filter-item') || plist.includes('filter-item'))){
                    $('.filter-menu').hide();
                }
            }catch(e){}
        });

		var vm = this, msgClickBtn = document.getElementById('messageCenterBtn'), msgContentDiv = document.getElementById('messsgeCenterBox');
        //定时器-新消息通知
        if(updateMessageCenterTimer){
            clearInterval(updateMessageCenterTimer);
        }
        updateMessageCenterTimer = setInterval(vm.getMsgCenterNews, '6000');

        //定时器-磁盘告警
        if(is_super_user == 'true'){
            if (diskSpaceAlarmInterval) {
                clearInterval(diskSpaceAlarmInterval);
            }
            diskSpaceAlarmInterval = setInterval(vm.getDiskSpaceInfo, '20000');
        }

		msgClickBtn.addEventListener('click', function(event){
			//vm.msgUrl = '${ctx}/msgCenter/queryMsgCenterPageList.action';
			vm.$refs.msgTable.refresh();
			//点击图标设置已读操作
			axios.post("${ctx}/msgCenter/updateMsgCenterLastReadTime.action",stringify({ })).then(function(response){
				var data = response.data;
				if(data["success"]){}else{}
			});
			//小红点及进行中的提示隐藏
			vm.circleTipShowOrHide = false;
			//vm.inprogressTipShowOrHide = false;
			//消息中心数据
			vm.messsgeCenterShowOrHide = true;
			event.stopPropagation()
		});
		document.addEventListener('click', function(event){
			vm.messsgeCenterShowOrHide = false
		});

		msgContentDiv.addEventListener('click', function(event){
			event.stopPropagation()
		});
	}
});

window.isNotSendLogout = true;
$.ajaxSetup({
	complete: function(req,status){
		if(req.status == 10001 || req.status == 401) {
			if(window.isNotSendLogout){
				window.isNotSendLogout = false;
				logout('${ctx}');
				if(top.logouto) top.logouto();
			}
		}
		//else if(req.readyState==0 && req.status == 0) toast('service not available',$('#mainpage'));
	}
});

    var isJumpToPage = '';
	//跨域问题菜单收起
	$(document).click(function(){
		parent.postMessage({msg:"mess"},"*");

		accessControl(accessJson);
	});

	// 防止高频重复点击
	var saveSelector = [
		'.linkbutton:contains(<%=rb.getString("QueDing")%>)',
		'.linkbutton:contains(<%=rb.getString("BaoCun")%>)',
		'.linkbutton:contains(<%=rb.getString("GO")%>)',
		'.el-button:contains(<%=rb.getString("QueDing")%>)',
		'.el-button:contains(OK)'
	];

	$(saveSelector.join(',')).each(function(idx,item){
		item.addEventListener('click',forbiddenButton,true);
	});

	$(document).ajaxSuccess(function(){
		accessControl(accessJson);
		$(saveSelector.join(',')).each(function(idx,item){
			item.addEventListener('click',forbiddenButton,true);
		});
		initSlideShadow();
		/* 新逻辑 -- 所有slider移动到#omc_app_ctn容器下 */
		var containers = ['.slidebarPanel'];
		containers.map(function(selector){
			var layers = document.querySelectorAll(selector);
			Array.from(layers).map(function(layer){
				var has = layer.getAttribute('ismoved');
				if(has!='yes') {
					document.querySelector('#omc_app_ctn').appendChild(layer);
					layer.setAttribute('ismoved','yes');
					$(layer).slideUp(function(){
						layer.style.visibility = 'visible';
					})
				}
			});
		})
	})
	//全局定时器，会在菜单点击时清楚
	var timer_reLoad;
	/*传递给后台的输入框的值  taskSearchText(任务列表) resultSearchText(结果列表)*/
	var taskSearchText = '';
	var queryStartTime = '';
	var queryEndTime = '';
	var resultSearchText = '';
	// 定时器-获取消息
	var getMsgInterval;
	//定时器-获取告警数量
	var alarmInterval;
	//定时器-获取告警声音和未读消息
	var alarmAlertInterval;
	var isUnread = true;
	/* 全局变量，记录当前活动告警数量*/
	var currAlarmCount = 0;
	//声音开关，默认开启，true-开启声音，false-关闭声音
	var isOpenSound = true;

	//用户会话超时时间
	var userSessionExpireTime = "";

	/* 标记提示资源阈值窗口是否打开 */
	var isWinResourceOverrangeOpen = false;
	var isCloudCore = "${Types}";

	//禁用F5刷新页面
	document.onkeydown=function(event){
		var e = event || window.event || arguments.callee.caller.arguments[0];
		if(e.keyCode==116){
			e.keyCode=0;
			e.returnValue = false;
		}
	}
	var lastVisitedTime = new Date(gloableTime);
    //记录未操作时间
    var noActionTime = new Date(gloableTime);
    var objTimer;
    //未操作时间定时器
    var noActionTimer;
    //SASprocedure页面中的两个定时器
	var SASProcressInterval;
	var datagridIntelVal;
	var eventBus;

    $(function(){
    	$('body').removeClass('loading');

    	localStorage.setItem("rsrp1",rsrp1);
    	localStorage.setItem("rsrp2",rsrp2);

    	//告警数量初始化赋值
    	//refreshAlarmStaticsCount();
    	//getMsg();

	eventBus = new Vue();
	
	// SSO 认证成功后清除锁定标志
	<c:if test="${ssoAuthSuccess == 'true'}">
	sessionStorage.locked = '';
	</c:if>
	
    	if(sessionStorage.locked) $("#winLockScreen").css("display","block");
		try {
			if(noLogoLockVue){
				noLogoLockVue.init();
			}
			if(baiCellsLockVue){
				baiCellsLockVue.init();
			}
		} catch (error) {}
    	//每27秒发送一次请求，用于后台判断用户是否关闭浏览器退出了系统
	  /*   setInterval(function(){
	    	var param = {};
	    	param["user_code"] = user_code;
	    	$.post("${ctx}/system/sysuser/inspectCloseExplorerEvent.action",param,"json")
	    },27000) ;*/
    	//右上角按钮鼠标悬停事件
        $("#mainpage").on("mouseenter mouseleave",".circleIcon .el-icon",function(){
        	$(this).siblings(".titleButtonText").stop();
        	$(this).siblings(".titleButtonText").fadeToggle();
        })

        $("#mainpage").on("focus blur",".queryGroup input",function(){
        	$(this).closest(".queryGroup").toggleClass('focus');
        })

        //Tab切换
        $("#mainpage").on("click",".tabsTitle > span",function(){
        	var tabText = $(this).text();
        	var tabClass = $(this).attr('tabtit');
        	var thisPar = $(this).closest('.panelDefault');

        	$(this).siblings("span").removeClass("active");
        	$(this).addClass("active");
        	$("." + tabClass).show().siblings("div").hide();
        	$(window).resize();

        	if(thisPar.find("div").hasClass('omcTabsPage_second')){
        		$("." + tabClass).find(".omcPageTitleContainer_second > li").first().click();
        	}
        })
        $.post("${ctx}/system/sysuser/checkLicenseWillExpire.action",{},function(data){
    	   if(data["success"]){
    		   if(data["message"] == ""){//license已经过期
    			   $.messager.alert(TiShi,"<%=rb.getString("LicenseYiGuoQiTiShi")%>")
    		   }else if(data["message"] == "macerror") {
  			   	   showMsg('alert_msg','<%=rb.getString("MacDiZhiCuoWu")%>')
    		   }else if(data["message"] == "uuiderror") {
  			   	   showMsg('alert_msg','<%=rb.getString("SystemUUIDCuoWu")%>')
    		   }else{
    			   showLicenseTip(data["message"])
    		   }
    	   }
       },"json")
       setInterval(function(){
    	  	var licenseTime = gloableTime.substring(11,16);
    	  	var second = gloableTime.substring(17)
    	  	if(licenseTime == "00:00" && (second - 0 >= 0) && (12-second >= 0)){
    	  		 $.post("${ctx}/system/sysuser/checkLicenseWillExpire.action",{},function(data){
    	      	   if(data["success"]){
    	      		   if(data["message"] != ""){//license已经过期
    	      			   if(data["message"] == "macerror") {
    	      			   	   showMsg('alert_msg','<%=rb.getString("MacDiZhiCuoWu")%>')
    	      		   	   }else if(data["message"] == "uuiderror") {
    	      			   	   showMsg('alert_msg','<%=rb.getString("SystemUUIDCuoWu")%>')
    	      		   	   }else {
        	      			   showLicenseTip(data["message"])
    	      		   	   }
    	      		   }
    	      	   }
    	         },"json")
    	  	}
       },6000)
       setInterval(function(){
    	   $.post("${ctx}/system/sysuser/checkLicenseWillExpire.action",{},function(data){
	      	   if(data["success"]){
	      		   if(data["message"] == ""){//license已经过期
	      			   $.messager.alert(TiShi,"<%=rb.getString("LicenseYiGuoQiTiShi")%>")
	      		   }else if(data["message"] == "macerror") {
      			   	   showMsg('alert_msg','<%=rb.getString("MacDiZhiCuoWu")%>')
      		   	   }else if(data["message"] == "uuiderror") {
      			   	   showMsg('alert_msg','<%=rb.getString("SystemUUIDCuoWu")%>')
      		   	   }
	      	   }
	         },"json")
       },3600000)
    // 初始化body的绑定事件，更新时间参数
       combonUpgradeToMsg();
    });

    function showLicenseTip(message){
    	var str = "<%=rb.getString("LicenseJiangGuoQi")%>"+message+"<%=rb.getString("QingShenQingXinDeLicense")%>";
	    $('.licenseTooltip .message').html(str)
		var width = $('.licenseTooltip').width();
		$('.licenseTooltip').css("left",'50%');
		var left = parseFloat($('.licenseTooltip').css("left")) - width/2;
		$('.licenseTooltip').css("left",left+'px');
		$('.licenseTooltip').animate({top:'20px'},200)
    }
  //菜单控制
    function passwdUpd() {
    	openDefaultWindow("${ctx}/system/sysuser/goModifyPwd.action",{
    		title: '<%=rb.getString("XiuGaiMiMaOpt")%>',
    		width:480,height:340
    	});
    }

    function changeOperator() {
    	openDefaultWindow("${ctx}/system/operator/toChangeOperatorPage.action",{
    		title: '<%=rb.getString("QieHuanYunYingShang")%>',
    		width: 520,
    		height: 550
    	});
    }

    function passwdWinCloseEvent(){
        var isFirstLogin = "${FirstLogin}";
        if (isFirstLogin == "true") {
            $.post("${ctx}/system/sysuser/queryPasswordRule.action", {}, function (data) {
                var messagerInfo = "<%=rb.getString("QingXiuGaiMiMa")%><%=rb.getString("JuHao")%>"  + "<br/>" +
                "<%=rb.getString("MiMaChangDu")%>" + "<%=rb.getString("MaoHao")%>" + data.MIN_LENGTH + "-" + data.MAX_LENGTH + "&nbsp;<%=rb.getString("ZiFu")%><%=rb.getString("DouHao")%>" +
                "<%=rb.getString("ZhangHuYouXiaoTianShu")%>" + "<%=rb.getString("MaoHao")%>"+ data.VALID_PERIOD + "&nbsp;<%=rb.getString("TianFuShu")%><%=rb.getString("JuHao")%>";
                $.messager.alert(TiShi, messagerInfo, "", function () {
                    /* $("#passwd").window("open");
                    $("#passwd").window("refresh", "${ctx}/system/sysuser/goModifyPwd.action"); */
                	openDefaultWindow("${ctx}/system/sysuser/goModifyPwd.action",{
	            		title: '<%=rb.getString("XiuGaiMiMaOpt")%>',
	            		width:480,height:340
	            	});
                });
            }, "json");
        }
    }

    <%-- 事件处理-数据表格加载失败 --%>
    function datagridLoadError() {
    	$.messager.alert(TiShi, "<%=rb.getString("JiaZaiShiBai")%>");
    }

    <%-- 事件处理-数据表格加载成功 --%>
    function datagridLoadSuccess() {
    	$(this).datagrid("fixRownumber");
    	$(this).datagrid("enableContextmenuAutoSize");
    }

    function getObjLength(obj){
        var count=0;
        for(var key in obj){
            if(key!= "contains" && key!="remove"){
                count += obj[key].length;
            }
        }
        return count;
    }
    //切换语言图标
    function ChangeLanguage(language){
    	$.ajax({
    		type:"Get",
    		url:"${ctx}/sys/login/changeLanguage.action?language_code="+language,
    		contentType:"application/json;charset=utf-8",
    		dataType:"json",
    		success:function(data){
    			window.location.href="${ctx}/sys/login/reloadAction.action?token=" + omctoken.replaceAll('+','%2B');
    		}
    	})
    }
    function closeNotice(){
        $(".NewFileTitInfo").hide();
    }
    function combonFirstTimeAndLeftMenu(){
    	//判断是否是第一次登录
        var isFirstLogin = "${FirstLogin}";
        var isModifyPwd = "${ModifyPwd}";

    	$.post("${ctx}/system/sysuser/queryPasswordRule.action", {}, function (data) {
    		userSessionExpireTime = data.user_session_expiration_min;
    		//判断是否是4A系统
            if (isCloudCore == "false") {
	            //会话超时时间不为空
	            if(userSessionExpireTime != "null" && userSessionExpireTime !=""){
	            	lastVisitedTime = new Date(gloableTime);
	                <%--定时执行函数判断会话是否超时--%>
	                objTimer = window.setInterval(checkSessionTime, 10000);
	            }
            }

	        if (isFirstLogin == "true" && isModifyPwd=="true") {
	        	var messagerInfo = "<%=rb.getString("QingXiuGaiMiMa")%><%=rb.getString("JuHao")%>"  + "<br/>" +
	            "<%=rb.getString("MiMaChangDu")%>" + "<%=rb.getString("MaoHao")%>" + data.MIN_LENGTH + "-" + data.MAX_LENGTH + "&nbsp;<%=rb.getString("ZiFu")%><%=rb.getString("DouHao")%>" +
	            "<%=rb.getString("ZhangHuYouXiaoTianShu")%>" + "<%=rb.getString("MaoHao")%>"+ data.VALID_PERIOD + "&nbsp;<%=rb.getString("TianFuShu")%><%=rb.getString("JuHao")%>";
	            $.messager.alert(TiShi, messagerInfo, "", function () {
	            	openDefaultWindow("${ctx}/system/sysuser/goModifyPwd.action",{
	            		title: '<%=rb.getString("XiuGaiMiMaOpt")%>',
	            		width:480,height:340
	            	});
	            });
	        }
    	}, "json");


    }
    function combonUpgradeToMsg(){
    	//判断是否存在版本升级通知
        $.post("${ctx}/system/sysuser/getUpgradeVersion.action", {}, function(data){
			try{
        		if(data && batchOperation) goVersionNotice(data);
			}catch(e){}
        }, "json");

        /*刷新当前告警数量*/
        refreshAliveAlarmCount();

        /*刷新北向接口通信异常告警 */
        // if ("${noItfnAlarmBtn}"== 0) {
		// 	refreshItfnAlarm();
        // }

        /* 给body元素绑定事件 */
        $("body").bind("mouseout", updateVisitedTime);
        $("body").bind("blur", updateVisitedTime);
        $("body").bind("focus", updateVisitedTime);
        $("body").bind("load", updateVisitedTime);
        $("body").bind("resize", updateVisitedTime);
        $("body").bind("scroll", updateVisitedTime);
        $("body").bind("unload", updateVisitedTime);
        $("body").bind("click", updateVisitedTime);
        $("body").bind("dbclick", updateVisitedTime);
        $("body").bind("mousedown", updateVisitedTime);
        $("body").bind("mouseup", updateVisitedTime);
        $("body").bind("mousemove", updateVisitedTime);
        $("body").bind("mouseover", updateVisitedTime);
        $("body").bind("mouseenter", updateVisitedTime);
        $("body").bind("mouseleave", updateVisitedTime);
        $("body").bind("change", updateVisitedTime);
        $("body").bind("select", updateVisitedTime);
        $("body").bind("submit", updateVisitedTime);
        $("body").bind("keydown", updateVisitedTime);
        $("body").bind("keypress", updateVisitedTime);
        $("body").bind("keyup", updateVisitedTime);
        $("body").bind("error", updateVisitedTime);

        /* 更新上次访问时间 */
        function updateVisitedTime() {
            lastVisitedTime = new Date(gloableTime);
            //记录自动退出登录时间
            noActionTime = new Date(gloableTime);
        }

        //定时更新为操作时间
        noActionTimer = window.setInterval(checkNoActionTime, 10000);

        //$("#winLockScreen").window("close");

        /* 解屏页面密码输入框回车事件 */
        $("#lockPassword").bind("keyup", function (event) {
            if (event.keyCode == 13) {
                cancelLock_RSA();
            }
        });

        // 显示当前运营商
        setCurrOperator("${SYSSESSIONKEY.operator_id}");

        // 定时查询服务器端消息，用于代替消息推送
        if (getMsgInterval) {
        	clearInterval(getMsgInterval);
        }
        if(alarmInterval){
        	clearInterval(alarmInterval);
        }
		if(alarmAlertInterval){
        	clearInterval(alarmAlertInterval);
        }
        //getMsg();
        getAlarmCount();
		getAlarmAlert();
		getMsgInterval = setInterval("getMsg()", getMsgTime);
        //alarmInterval = setInterval("getAlarmCount()", getMsgTimeAlarm);
    }
    function dowloadFailureFileBatch(){
    	$("#downloadFailureBatchForm").form("submit");
    	$('#winDowloadFailureFileBatch').window('close');
    }
</script>

<jsp:include page="<%=subScriptPage %>" flush="true"/>
</body>
</html>
