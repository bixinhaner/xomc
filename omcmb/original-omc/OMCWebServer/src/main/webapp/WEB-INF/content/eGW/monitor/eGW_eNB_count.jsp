<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
.enbCountDiv{
	width : 45%;
	height : 99%;
	display:inline-block;
	overflow-y:auto;
}
.enbCountChartStyle{
	margin-left:20px;
	width: 96%;
	height:300px;
}
.enbCountChartTitle{
	display:inline-block;
	font-weight:410;
	font-size: 14px;
	color: #0f344d;
}
.rightHideDiv{
	position:absolute;
	width:900px;
	background:#FFFFFF;
	right:-2000px;	
	top:0px;
	bottom:0px;
	z-index:100;
	border:1px solid #EEE;
}
#newTemplateDiv_header,.viewTemplateDiv_header{
	position:absolute;
	width:100%;
	line-height:50px;
	height:50px;
	top:0px;
	color:#7993B6;
	font-size:16px;
}	
#newTemplateDiv_body,.viewTemplateDiv_body{
	position:absolute;
	width:100%;
	top:50px;
	overflow:auto;
	bottom:10px;
}
</style>
<div class="panelDefault" style="border:none;">
	<div class="slidebarTitleDiv">
    	<span><%=rb.getString("JiZhanSheBei")%></span>
	</div>
	<div class="omcTitleButtonGroup" style="top:10px;">
		<div class="omcTitleButtonGroupItem">
			<span class="titleButtonText"><%=rb.getString("DaoChu")%></span>
			<span class="el-icon el-icon-circle-export" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="exportENBDetail()">
			</span>
		</div>
		<div class="omcTitleButtonGroupItem">
			<span class="titleButtonText"><%=rb.getString("GuanBi")%></span>
			<span class="el-icon el-icon-circle-close" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="closeENBCountDetailPage()"></span>
		</div>
	</div>
	<div class="singleContentDiv" style="display:flex;">
		<div id="enbCountDatagrid" class="enbCountDiv" >
			<table id="eNBCount_datagrid" class="panelTableDiv"></table>
		</div>
		<div class="enbCountDiv" style="border-left:2px solid #E9E9E9;flex-grow:1;padding:20px;">
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("UEShu")%></span>
			</div>
			<div id="eNBcountChart" class="enbCountChartStyle"></div> 
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("EGWShangXingLiuLiang")%></span>
			</div>
			<div id="upTrafficChart" class="enbCountChartStyle"></div> 
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("EGWXiaXingLiuLiang")%></span>
			</div>
			<div id="dowmTrafficChart" class="enbCountChartStyle"></div> 
		</div>
	</div>
</div>
<div id="eNBCountDetailPanel" class="rightHideDiv slidebarPanel"></div>
<!-- 用户工具栏 -->
<div id="toolbar_eNBCount_datagrid" class="toolbarContainer">
    <div class="queryGroup">
        <input id="eNBCount_datagrid_Input" name="search_text" style="margin-left:0px;width:330px;" placeholder="ECI / <%=rb.getString("EnodebId")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
        <b class="el-icon el-icon-common-search" onclick='$("#eNBCount_datagrid").datagrid("reload")'></b>
    </div>
</div>
<script>
var datagridRow = $("#eGW_monitor_datagrid").datagrid("getSelected");
var gwIp = datagridRow.gw_ip;
var gwPort = datagridRow.gw_port;
var reportCycle = 10;
var startTime = formatDate(new Date(gloableTime)).substring(0,10)+' 00:00:00';
$(function(){
   closeLoading();
   $("#eNBCount_datagrid").datagrid({
	   url:'${ctx}/egw/monitor/queryEnbMonitorPageList.action',
	   border: false,
	   fit: true,
	   fitColumns: true,
	   rownumbers:true,
	   singleSelect: false,
	   idField: 'cell_id',
	   toolbar:'#toolbar_eNBCount_datagrid',
	   pagination: true,
	   pagePosition: 'bottom',
	   checkOnSelect: true,
	   selectcOnCheck: true,
	   onBeforeSelect: onBeforeSelectFun,
	   onBeforeCheck: onBeforeSelectFun,
	   onCheck: checkedChangeFun,
	   onUncheck: unCheckedChangeFun,
	   columns: [[
			{field: 'ck',checkbox:true, width: 100},
			{field: 'cell_id',sortable:true, width: 80, title: 'ECI' },
			{field: 'enb_id',sortable:true,width: 100, title: '<%=rb.getString("EnodebId")%>'},
			{field: 'ip', sortable:true,width: 100, title: 'IP'},
			{field: 'status',sortable:true,width: 80, title: '<%=rb.getString("ZhuangTai")%>'},
			{field: 'uptime',width:120,title:'<%=rb.getString("ShangXiaShiJian")%>'}
		]],
        onBeforeLoad : beforeLoad_eNBCount_datagrid,
		onLoadSuccess: loadSuc_eNBCount_datagrid  
   });
   $("#eNBCount_datagrid_Input").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#eNBCount_datagrid").datagrid("reload");
			$("#eNBCount_datagrid_Input").blur();
		}	
	});
   goSetChartData('eNBcountChart',"",[],'we','<%=rb.getString("ueGe")%>');
   goSetChartData('upTrafficChart',"",[],'we','MB');
   goSetChartData('dowmTrafficChart',"",[],'we','MB');
})
function beforeLoad_eNBCount_datagrid(param){
	param["gwIp"] = gwIp;
	param["gwPort"] = gwPort;
	param["timeZone"] = timeZone;
	var search_text = $("#eNBCount_datagrid_Input").val();
	if(search_text != ""){
		param["searchText"] = search_text;
	}
}
//点击前判断是否>=5，如果是就不能选中  遍历是为了把原来选中的选中，否则会出现一个不选的情况
function onBeforeSelectFun(index, row){
 	var selectedNum = $("#eNBCount_datagrid").datagrid("getChecked").length;
	if(selectedNum>=5){
		var isIn = false;
		$.each(allSelectedId,function(index,item){
			if(row.cell_id == item) isIn = true;
		});
		if(!isIn){
 			return false;
		} 
	}
}
/**
* 表格勾选事件
* @param index{number} 数组下标
* @param row{object}  行数据
*/
function checkedChangeFun(index, row){
	var selectCheckNum = $("#eNBCount_datagrid").datagrid("getChecked").length;
	if (selectCheckNum > 4){
   		$("#eNBCountDetailPage input[type='checkbox']:not(:checked)").prop("disabled",true);
    } else {
    	$("#eNBCountDetailPage input[type='checkbox']:not(:checked)").prop("disabled",false);        	 
    }
	getEGWChartData();
}
/**
* 表格取消勾选事件
* @param index{number} 数组下标
* @param row{object}  行数据
*/
var currentTimer,time;
function unCheckedChangeFun(index, row){
	/****************控制高频触发******************/
	var now = new Date(gloableTime).getTime();
	if(time){
		if(now-time<100){
		  clearTimeout(currentTimer);
		}
	}else{
		time = now;
	}
	/*************************************/
	currentTimer = setTimeout(function(){
		var selectUncheckNum = $("#eNBCount_datagrid").datagrid("getChecked").length;
		if (selectUncheckNum > 4){
		  	$("#eNBCountDetailPage input[type='checkbox']:not(:checked)").prop("disabled",true);
		} else {
		   	$("#eNBCountDetailPage input[type='checkbox']:not(:checked)").prop("disabled",false);        	 
		}
		var isSplit = false;
		$.each(allSelectedId,function(index,item){
			if(row.cell_id == item) isSplit = index;
		});
		if(isSplit>-1){
			allSelectedId.splice(isSplit,1);
		} 
		getEGWChartData();
		time=0;// 计时回到起点
	},200);
}
var allSelectedId = []; //选中行的主键

//加载成功后1、判断是否可继续勾选 2、将之前选中的重新勾选
function loadSuc_eNBCount_datagrid(data){
	$(this).datagrid("enableContextmenuAutoSize");
 	$("#eNBCount_datagrid").parent().find(".datagrid-header-check").find("input").attr("style","display:none;");
 	var selectedNum = $("#eNBCount_datagrid").datagrid("getSelections").length;
 	var allSelectedRows = $("#eNBCount_datagrid	").datagrid("getSelections");
 	var allRows = $("#eNBCount_datagrid").datagrid("getRows");
	allSelectedId = [];
	if(allSelectedRows.length == 0 && allRows.length>0 ){
		allSelectedId.push(allRows[0].cell_id);
	}else{
	 	for(var i=0;i<allSelectedRows.length;i++){
	 		allSelectedId.push(allSelectedRows[i].cell_id);
	 	}
	}
 	for(var i=0;i<allRows.length;i++){
 		if(allSelectedId.includes(allRows[i].cell_id)){
 			$("#eNBCount_datagrid").datagrid("checkRow",i);
 		}else{
 			$("#eNBCount_datagrid").datagrid("uncheckRow",i);
 		}
 	}
 	if (selectedNum > 4){
   		$("#eNBCountDetailPage input[type='checkbox']:not(:checked)").prop("disabled",true);
    } else {
    	$("#eNBCountDetailPage input[type='checkbox']:not(:checked)").prop("disabled",false);        	 
    }
}
// 关闭基站数（在线）页面
function closeENBCountDetailPage(){
	
	$("#eNBCountDetailPage").slideUp(500,function(){
		$("#eNBCountDetailPage").html("");
	});
}

function goSetChartData(chart_id,lineCode,chart_data,name,unit) {
    //初始化节点数据
    var timesCount = 24*(60/reportCycle)+1 ; // 6*(60/15) hours
    var timesDataArr = [];
    if(chart_data){
    	var series_data = [];
		   startTimeStr = startTime;
	       for(var timesIndex = 0; timesIndex<timesCount; timesIndex++ ){
	    	   timesDataArr.push(formatDate(addTimes(new Date(startTimeStr),timesIndex*reportCycle)));
	       }
			
	       for (var code_index = 0; code_index < lineCode.length; code_index++) {
	           var seriesEle_data = [];
	           
	           for(var timesIndex = 0; timesIndex<timesCount; timesIndex++ ){
	        	   seriesEle_data.push('-');
		       }
	          seriesEle_data = chart_data[lineCode[code_index]];
	           var seriesEle = {
	               name : lineCode[code_index],
	               type : 'line',
	               symbolSize : 1,
	               showAllSymbol : true,
	               data : seriesEle_data
	           };
	           series_data.push(seriesEle);
	       }
	       
	       var title_obj = {
	           code : lineCode,
	           name : name,
	           unit : unit
	       };
	       var legend_obj = {
	           code : lineCode,
	           name : lineCode
	       };
	       
	       var chart_data = {
	           title_data : title_obj,
	           legend_data : legend_obj,
	           x_data : timesDataArr,
	           series_data : series_data
	       };
	       try{
		       goCreateChart(chart_id, chart_data);
	       }catch(e){}
    } 	
 }
 //创建图表
 function goCreateChart(elementId, chart_data) {
 	var kpiChart = echarts.init(document.getElementById(elementId));
	if(kpiChart) kpiChart.clear();
     var option = {
         title : {
         	codedata : chart_data.title_data.code,
             x : 'left',
             y : -10,
             subtextStyle : {
                 fontSize : 12,
                 color : '#CAC5D4',
             }
         },
	        color : ['#85b1de','#69e7e7','#97dbb9','#e9a4a4','#bf8bf3'],
         tooltip : {
             trigger : 'axis'
         },
         grid : {
         	top : '72px',
         	bottom : '40px',
         	left : '50px',
         	right : '50px'
         },
         legend : {
	              data : [{
		            	  	name : chart_data.legend_data.name[0]?chart_data.legend_data.name[0]:chart_data.legend_data.code[0],
		            	  	textStyle : {
	          	  			color : '#85b1de'
	            	  	}
	            	  },{
		            	  	name : chart_data.legend_data.name[1]?chart_data.legend_data.name[1]:chart_data.legend_data.code[1],
		            	  	textStyle : {
		            	  		color : '#69e7e7'
		            	  	}
	            	  },{
		            	  	name : chart_data.legend_data.name[2]?chart_data.legend_data.name[2]:chart_data.legend_data.code[2],
		            	  	textStyle : {
		            	  		color : '#97dbb9'
		            	  	}
		              },{
		            	  	name : chart_data.legend_data.name[3]?chart_data.legend_data.name[3]:chart_data.legend_data.code[3],
		            	  	textStyle : {
		            	  		color : '#e9a4a4'
		            	  	}
		              },{
		            	  	name : chart_data.legend_data.name[4]?chart_data.legend_data.name[4]:chart_data.legend_data.code[4],
		            	  	textStyle : {
		            	  		color : '#bf8bf3'
		            	  	}
		              }
	              ]
	          }, 
         xAxis : [{
                     type : 'category',
	                    boundaryGap : false,
                     data : chart_data.x_data,
                     axisLabel : {
                         formatter : function(val) {
                         	var secondTime = val.split(' ')[1];
                         		clock = secondTime.substring(0,2);
	                        	val = secondTime.substring(0,5);
                             if(secondTime.substring(3,5)=='00') return clock;
                             return val;
                         },
                         interval:function(index){
                         	if(index%(60/reportCycle)==0 && index!=24*(60/reportCycle)){
                         		return true;
                         	}
                         }
                     },
                     splitLine : {
                         show : false
                     },
                     name : '<%=rb.getString("XiaoShi")%>'
                 }],
         yAxis : [{
                     name: "<%=rb.getString("ZuoKuoHao")%>" + chart_data.title_data.unit + "<%=rb.getString("YouKuoHao")%>",
                     minInterval: (elementId=='eNBcountChart')?1:null,
                     type : 'value',
                     axisLabel : {
                        formatter : '{value}'
                     }
                 }],
         series : chart_data.series_data
     };
     kpiChart.setOption(option);
	 $(window).on('resize',function(){
		setTimeout(function(){
			kpiChart.resize();
		},200);
	 })
 }
function exportENBDetail(){
	$("#eNBCountDetailPanel").animate({right:'0px'},500);
	$("#eNBCountDetailPanel").panel({
		 width:900,
		 href:'${ctx}/egw/monitor/toMonitorEnbExport.action',
		 onLoad:function(){
			 //$("#viewTemplateTitle").html("<%=rb.getString("MuBanXiangQing")%>").attr("tempId",data.temp_id);
		 }
	})
}
var startTime = formatDate(new Date(gloableTime)).substring(0,10)+' 00:00:00';
//获取图表数据
function getEGWChartData(){
	var cellIds = [];
	var selectedENBData= $("#eNBCount_datagrid").datagrid("getChecked");
	
	$.each(selectedENBData,function(index,ele){
		cellIds.push(ele.cell_id)
	})
	var lineCode = cellIds;
	cellIds =cellIds.join(",") ;
	var endTime = formatDate(addDate(new Date(startTime), 1));;
	var params = {
			"gwIp" : gwIp,
			"gwPort" : gwPort,
			"cellIds" : cellIds,
			"startTime" : startTime,
			"endTime" : endTime,
			"timeZone" : timeZone
	}                                                                                                  
	$.post("${ctx}/egw/monitor/getEnbStatisticsListChart.action",params,function(data){
        if (data.ueCount) {
        	goSetChartData('eNBcountChart',lineCode,data.ueCount,'we','<%=rb.getString("ueGe")%>');
        } else {
        	goSetChartData('eNBcountChart',lineCode,[],'we','<%=rb.getString("ueGe")%>');
        } 
        if (data.ul) {
        	goSetChartData('upTrafficChart',lineCode,data.ul,'we','MB');
        } else {
        	goSetChartData('upTrafficChart',lineCode,[],'we','MB');
        } 
        if (data.dl) {
        	goSetChartData('dowmTrafficChart',lineCode,data.dl,'we','MB');
        } else {
        	goSetChartData('dowmTrafficChart',lineCode,[],'we','MB');
        } 
    }, "json");
}
</script>