<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<!-- 新文件版本升级通知 -->
<div class="NewFileTitInfo" style="z-index:2001">
	<div style="display:inline;">
		<div class="NewFileTitInfoCont"></div>
		<span id="closeTit" class="closeTit" onclick="closeNotice()"></span>
		<img id="briefInfo" style="display:none;margin-right:30px;margin-top:12px;float:right;cursor:pointer;" src="${ctx}/css/images/bi/upArrow.png"/>
	</div>
   	<div id="NewFileDetials" style="display:none; max-height:600px;overflow:auto;top:30px;width: 900px; position: absolute;margin:10px auto;background: #cdf4f4; z-index: 1002; padding:10px 0px;">
   	</div>
</div>
<div id="mainCover" class="window-mask" style="display:none;width:2000px;height:1800px;position:absolute;left:0px;top:0px;z-index:1000"></div>
<div id="modal" class="modal"></div>




<%---锁屏界面--%>
<div class="lockbody" id="winLockScreen" style="display:none;">
	<div class='lockIconItem'>
   		<div style="margin:auto;display:flex">
   			<p style='line-height:40px;'><span class="el-icon el-icon-circle-warning" style="margin-right:10px;font-size:16px;"></span><span style="font-size:14px;"><%=rb.getString("JianRongTiShi")%>:</span></p>
   			<div style="display:inline-block;margin-left:10px">
   				<span class="browser_icon google_icon"></span>
   				<span class="browser_icon safari_icon"></span>
   				<span class="browser_icon firefox_icon"></span>
   				<span class="browser_icon edge_icon"></span>
   			</div>
   		</div>
    </div>
	<div class="lockmain">
		<!-- <div class="locklogo">北京融昱信息技术有限公司</div> -->
		<div class="lockformbox">
			<form id="lockform"  style="width: 100%; height: 100%; text-align: center; padding-right: 5px;">  
				<input type="hidden" name="userInfo.logintype" value="0" />
				<div class="lock_bar_logo_div"></div>
				<div class="lock_user_div">
					<div class="input_ico_user_div_lock"></div>
					<input style="color:#96b3ed" readonly type="text" id="uid_lock" class="input_" name="userInfo.usercode"  value="${userInfo.usercode}">
				</div>
				<div class="lock_pwd_div">
					<div class="input_ico_pwd_div_lock"></div>
					<input type="password" style="display:none">
					<input id="lockPassword" type="password" class="input_" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>">
					<span id="lockPassResult" style="color:red; font-size: 13px; display:block;text-align:left;margin-left:41px;margin-top:8px;"></span>
				</div>
				<div class="lock_btn_div" onclick="javascript:cancelLock()">
					<span style="line-height: 38px;	font-size: 20px; color: #FFFFFF;"><%=rb.getString("JieSuo")%></span>
				</div>
			<form>
		</div>
	</div>
</div> 

<script type="text/javascript">

    $(function () {
    	//扩展easyui控件
    	extendEasyui();
    	
        combonFirstTimeAndLeftMenu();
		
        <%-- 是否有密码将过期提醒 --%>
         //判断是否是4A系统
        if (isCloudCore == "false") {
	        $.post("${ctx}/system/sysuser/queryPaswdExpirePrompt.action", {}, function (data) {
	            if (data["Expire"].length > 0) {
	                $.messager.alert(TiShi, data["Expire"]);
	                return;
	            }
	        }, "json");
        }
        
      //判断是否是兼容浏览器
        var explorer = window.navigator.userAgent;
		var explorFlag = false;
		if(explorer.indexOf("Firefox") > 0){//火狐浏览器
			explorFlag = true;
		}else if(explorer.indexOf("Edg") > 0){//Edge浏览器
			explorFlag = true;
		}else if(explorer.indexOf("Mac") > 0){//Safari浏览器
			explorFlag = true;
		}else if(isChrome()){//Google浏览器
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
			$(".lockIconItem").hide();
		}else{
			$(".lockIconItem").show();
		}      
    });
    function isChrome(){
		var agent = window.navigator.userAgent,
			reg = /^[\s\S]+Chrome\/[0-9\. ]+Safari\/[0-9\.]+$/;
	
		return reg.test(agent);
	}
    function goVersionNotice(data){
    	$(".NewFileTitInfo").hide();
        if (!data || getObjLength(data)<=0){
            return;
        }
        
        // enodeb（基站全量升级）、enodeb_patch（基站补丁升级）、enodeb_uboot（基站UBOOT升级）、cpe_odu（CPE ODU升级）、cpe_idu（CPE IDU升级）
        var enodeb_upgrade = data.enodeb_upgrade;
        var enodeb_patch_upgrade = data.enodeb_patch_upgrade;
        var enodeb_uboot_upgrade = data.enodeb_uboot_upgrade;
        var cpe_odu_upgrade = data.cpe_odu_upgrade;
        var cpe_idu_upgrade = data.cpe_idu_upgrade;
        
        var notice = {};
        if (enodeb_upgrade){
            notice["enodeb"] =[];
            for(var i=0;i<enodeb_upgrade.length;i++){  
                var version = {
                        version : enodeb_upgrade[i],
                        notice : "--"+'<%=rb.getString("Msg_YouJiZhanRuanJianBanBenKeGengXin")%>' + enodeb_upgrade[i] + '<%=rb.getString("Msg_KeYongQingShengJi")%>'
                };
                notice["enodeb"].push(version);
            }
        }
        if (enodeb_patch_upgrade){
            notice["enodeb_patch"] = [];
            for(var i=0;i<enodeb_patch_upgrade.length;i++){ 
                var version = {
                        version : enodeb_patch_upgrade[i],
                        notice : "--"+'<%=rb.getString("Msg_YouJiZhanPATCHRuanJianBanBenKeGengXin")%>' + enodeb_patch_upgrade[i] + '<%=rb.getString("Msg_KeYongQingShengJi")%>'
                };
                notice["enodeb_patch"].push(version);
            }
        }
        if (enodeb_uboot_upgrade){
            notice["enodeb_uboot"] = [];
            for(var i=0;i<enodeb_uboot_upgrade.length;i++){
                var version = {
                        version : enodeb_uboot_upgrade[i],
                        notice : "--"+'<%=rb.getString("Msg_YouJiZhanUBOOTRuanJianBanBenKeGengXin")%>'+enodeb_uboot_upgrade[i] + '<%=rb.getString("Msg_KeYongQingShengJi")%>'
                };
                notice["enodeb_uboot"].push(version);
            }
        }
        if (cpe_odu_upgrade){
            notice["cpe_odu"] = [];
            for(var i=0;i<cpe_odu_upgrade.length;i++){
                var version = {
                        version : cpe_odu_upgrade[i],
                        notice : "--"+'<%=rb.getString("Msg_YouCPERuanJianBanBenKeGengXin")%>'+cpe_odu_upgrade[i] + '<%=rb.getString("Msg_KeYongQingShengJi")%>'
                };
                notice["cpe_odu"].push(version);
            }
        }
        if (cpe_idu_upgrade){
            notice["cpe_idu"] = [];
            for(var i=0;i<cpe_idu_upgrade.length;i++){
                var version = {
                        version : cpe_idu_upgrade[i],
                        notice : "--"+'<%=rb.getString("Msg_YouCPERuanJianBanBenKeGengXin")%>'+cpe_idu_upgrade[i] + '<%=rb.getString("Msg_KeYongQingShengJi")%>'
                };
                notice["cpe_idu"].push(version);
            }
        }
        if (getObjLength(notice)>0){
        	$(".NewFileTitInfo").show();
            if (getObjLength(notice)>1){ 
                $(".NewFileTitInfoCont").append("<span id='infoTitle' style='color:#333333;font-size:14px;'><%=rb.getString("YouKeShengJiWenJianDianJiChaKanXiangQing")%>.</span>");
                $(".NewFileTitInfoCont").append('<img id="moreInfo" style="margin-top:6px;margin-left:15px;cursor:pointer;" src="${ctx}/css/images/bi/downArrow.png"/>');
                $(".NewFileTitInfo").append('<span class="noNotice" onclick="NoNotice()"><%=rb.getString("Button_HuLueSuoYouBanBen")%></span>');
                for(var key in notice){
                	if(typeof notice[key] != 'function'){  
                        for(var i=0;i<notice[key].length;i++){
                            var paramObj = {type:key,version: notice[key][i].version,versionId:notice[key][i].versionId};
                            var infoDetail = "<div style='margin:0px 40px;'>";
                            infoDetail+="<span style='line-height:40px;color:#6D7A7D;font-size:14px;'>" + notice[key][i].notice +"</span>";
                            infoDetail+="<span class='noNotice' onclick='NoNotice(this,"+ JSON.stringify(paramObj)+ ")'>"+"<%=rb.getString("Button_HuLue")%>"+"</span>";
                            infoDetail+="</div>";
                            $("#NewFileDetials").append(infoDetail);
                        }
                    }
                }
                
                $("#moreInfo").click(function(){//Update information for details:
                    $("#mainCover").show();
                    $("#NewFileDetials").slideDown(300);    
                    $("#moreInfo").hide();
                    $("#closeTit").hide();
                    $("#briefInfo").show();
                    $("#infoTitle").html("<%=rb.getString("XiangXiXinXiRuXia")%>:");
                })
                $("#briefInfo").click(function(){
                    $("#mainCover").hide();
                    $("#NewFileDetials").slideUp(300);
                    $("#moreInfo").show();
                    $("#closeTit").show();
                    $("#briefInfo").hide();
                    $("#infoTitle").html("<%=rb.getString("YouKeShengJiWenJianDianJiChaKanXiangQing")%>.");
                })
            } else {
                for (var key in notice){
                	if(typeof notice[key] != 'function'){     
                        for(var i=0;i<notice[key].length;i++){                          
                            $(".NewFileTitInfoCont").text(notice[key][i].notice );
                            var paramObj = {type:key,version: notice[key][i].version,versionId:notice[key][i].versionId};
                            $(".NewFileTitInfo").append("<span class='noNotice' onclick='NoNotice(this," + JSON.stringify(paramObj) + ")'><%=rb.getString("Button_HuLueCiBanBen")%></span>");
                        }
                    }
                }
            }
        }
    }
    
    function NoNotice(ele,obj){
    	//注意upgrade_type枚举：enodeb（基站全量升级）、enodeb_patch（基站补丁升级）、enodeb_uboot（基站UBOOT升级）、cpe_odu（CPE ODU升级）、cpe_idu（CPE IDU升级）
    	//version版本号
    	if (obj===undefined){
	    	$.post("${ctx}/system/sysuser/goCloseUpgradePrompt.action", {}, function(data){
	    		if(data.success === true){	    			
			    	$(".NewFileTitInfo").hide();
	    		}
	    	}, "json");
    	} else {
	    	$.post("${ctx}/system/sysuser/goCloseUpgradePrompt.action", {upgrade_type:obj.type,version:obj.version,versionId:obj.versionId}, function(data){
	    		if(data.success === true){
		    		$($(ele).parent()).remove();	    			
	    		}
	    	}, "json");
    	}
    }
    
</script>
</html>