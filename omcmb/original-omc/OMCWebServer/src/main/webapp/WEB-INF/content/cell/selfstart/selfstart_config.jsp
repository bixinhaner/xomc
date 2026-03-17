<%--
  User: Yujianfei
  Date: 2016/1/23
  Time: 11:15
  To change this template use File | Settings | File Templates.
--%>
<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
  .itemDiv {
      height: 30px;
      padding: 5px;
  }
  .itemDiv span {
    display: inline-block;
    width: 200px;
  }
  .item {
    width: 250px;
  }
</style>
<div class="easyui-layout" data-options="border:false,fit:true">
  <div region="center" data-options="border:false">
      <br/>
      <div class="itemDiv">
        <span><%=rb.getString("XIAOQUID")%></span>
        <input type="text" name="LTE_CELL_ECI" class="border border-box item" value="${CELL_ID}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("PCI")%></span>
        <input type="text" name="LTE_PHY_CELLID_LIST" class="border border-box item" value="${PCI}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("XiaXingPinDian")%></span>
        <input type="text" name="LTE_DL_EARFCN" class="border border-box item" value="${DL_EARFCN}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("ShangXingPinDian")%></span>
        <input type="text" name="LTE_UL_EARFCN" class="border border-box item" value="${UL_EARFCN}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("TAC")%></span>
        <input type="text" name="LTE_TAC" class="border border-box item" value="${TAC}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("PLMN")%></span>
        <input type="text" name="LTE_OAM_PLMNID" class="border border-box item" value="${PLMN}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("PA")%></span>
        <input type="text" name="LTE_PA" class="border border-box item" value="${PA}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("PB")%></span>
        <input type="text" name="LTE_PB" class="border border-box item" value="${PB}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("CanKaoXinHaoQiangDu")%></span>
        <input type="text" name="LTE_REFERENCE_SIG_POWER" class="border border-box item" value="${referenceSigPower}"/>
        <br/>
      </div>
      <div class="itemDiv">
        <span><%=rb.getString("MMEDIZHI")%></span>
        <input type="text" name="LTE_SIGLINK_SERVER_LIST" class="border border-box item" value="${MME_ADDRESS}"/>
        <br/>
      </div>
    </div>
    <div region="south" data-options="border:true" style="border-width:1px 0 0 0;height:37px;padding:5px 0;">
        <a class="easyui-linkbutton" onclick="closeSelfWin()" style="float:right;margin-right:10px;"><%=rb.getString("QuXiao")%></a>
        <a class="easyui-linkbutton" onclick="selfstartParamsCommit()" style="float:right;margin-right:10px;"><%=rb.getString("QueDing")%></a>
    </div>
</div>

<script type="text/javascript">
    <%-- 关闭窗口 --%>
    function closeSelfWin() {
        /* $("#winSelfstartConfigTab").window("close"); */
    	closeDefaultWindow();
    }

    function selfstartParamsCommit(){
        var params={};
        params["cell"] = "<%=request.getParameter("smallCellCode") %>";
        $(".item").each(function() {
            if ($(this).val()) {
                params[$(this).attr("name")] = $(this).val();
            }
        });
        $.post("${ctx}/cell/selfstart/selfstartConfig.action", params, function(data) {
            if (data["success"]) {
                closeSelfWin();
                $.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("CaoZuoChengGong")%>");
            } else {
                $.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
            }
        }, "json");
    }
</script>