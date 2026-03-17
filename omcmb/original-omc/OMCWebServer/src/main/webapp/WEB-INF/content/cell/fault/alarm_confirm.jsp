<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
    .confirmInfo {
        height: 60px;
        line-height: 16px;
    }

    .confirmInfo span {
        display: inline-block;
        width: 100px;
        vertical-align: top;
    }

    .confirmInfo input {
        width: 250px;
    }
</style>

<div class="easyui-panel" data-options="border:false,fit:true">
    <input type="hidden" id="confirmAlarmType" value="${type}"/>
    <input type="hidden" id="confirm_alarm_id" value="${alarm_id}"/>

    <div class="easyui-layout" data-options="fit:true">
        <div region="center" data-options="border:false" style="padding: 20px;">
            <div class="confirmInfo not-confirm-hide">
                <span><%=rb.getString("QueRenRen")%></span>
                <input type="text" class="border border-box" disabled="disabled" value="${confirmInfo.DEAL_USER}"/>
            </div>
            <div class="confirmInfo not-confirm-hide">
                <span><%=rb.getString("QueRenShiJian")%></span>
                <input type="text" class="border border-box" disabled="disabled" value="${confirmInfo.DEAL_TIME}"/>
            </div>
			<div class='confirmInfo clearInfo'>
				<%=rb.getString("QueRenQingChuGaoJing")%>
			</div>
            <div class="confirmInfo">
                <span><%=rb.getString("MiaoShu")%></span>
                <textarea id="txtAlarmConfirm" class="border border-box" style="width:250px;height:105px;resize:none;"
                          maxlength=500
                          placeholder="<%=rb.getString("QueRenXinXi")%>">${confirmInfo.DEAL_MEMO}</textarea>
            </div>
        </div>
        <div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
        	<div class="windowButtonGroup" style="margin-right:25px">
        		<a href="#" class="el-button el-button--primary" onclick="alarmConfirm()"><%=rb.getString("QueRen")%></a>
            	<a href="#" class="el-button" onclick="javascript: closeDefaultWindow();"><%=rb.getString("QuXiao")%></a>
        	</div>
        </div>
    </div>
</div>

<script type="text/javascript">
    $(function () {
    	if(clearAlarmFlag){
    		//说明是清除告警
    		$('.clearInfo').show();
    		$(".not-confirm-hide").hide();
    		setDefaultWindow({height: 350});
    	}else{
    		$('.clearInfo').hide();
    		if ("${confirmInfo.DEAL_STATE}"=="0" || "${confirmInfo.DEAL_STATE}"=="2") {
                // 此条告警尚未确认，隐藏确认人、确认时间，窗口高度调小
                $(".not-confirm-hide").hide();
                setDefaultWindow({height: 250});
            } else {
            	setDefaultWindow({height: 450});
            }
    	}
    });

    // 确认告警
    function alarmConfirm() {
        var type = $("#confirmAlarmType").val();
        if (type == 'active') {
            grid = $("#gridAlarm");
            infogrid = $("#enbActiveAlarm");
        } else if (type == 'history') {
            grid = $("#gridAlarmHis");
            infogrid = $("#enbHistoryAlarm");
        }
      	var url,
      		msg;
      	if( clearAlarmFlag == true){
			url = '${ctx}/cell/fault/clearAlarm.action';    //清除告警 
			msg = '<%=rb.getString("QingChuGaoJingTiShi")%>'
		}else {
			url = '${ctx}/cell/fault/confirmAlarm.action';     //确认告警 
			msg = '<%=rb.getString("QueRenGaoJingTiShi")%>'
		}
      	var gridData = infogrid.datagrid('getSelected');
      	var params = {
      			alarm_id : $("#confirm_alarm_id").val(),
      			small_cell_code : gridData.small_cell_code,
      			type : type,
      			text : $('#txtAlarmConfirm').val()
      			
      	}
        $.post(url, params, function(data) {
            if (data["success"]) {
              	showMsg('success_msg',msg + '<%=rb.getString("TiShiChengGong")%>');
                closeDefaultWindow();
                grid.datagrid("load");
                infogrid.datagrid("load");
            } else {
            	showMsg('error_msg',data["message"]);
            }
        }, "json");
    }
</script>