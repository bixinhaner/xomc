<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	/* #viewUserInfo{
		width:100%;
		height:100%;
	} */
	#viewUserInfo .userInfo{
		margin-bottom:30px;
	}
	#viewUserInfo .userInfo span:nth-child(1){
		font-size:14px;
		color:#999;
		display:inline-block;
		width:100px;
		text-align:right;
	}
	#viewUserInfo .userInfo span:nth-child(2){
		font-size:14px;
		color:#333;
	}
	#viewUserInfo .userInfo .groupInfo{
		background:#F7F7F7;
		padding:0 10px;
		border-radius:10px;
		font-size:12px !important;
		display:inline-block;
		height:20px;
		line-height:20px;
		border:1px solid #E8EAEC;
		margin-right:10px;
		margin-bottom:5px;
	}
</style>
<div id='viewUserInfo'>
	<div style='width:100%;display:flex;flex-direction:row;'>
		<div style='flex:1;height:120px;margin-top:10px;height:auto;'>
			<p class='userInfo'>
				<span><%=rb.getString("YongHuMingCheng")%>:&nbsp;</span>
				<span>{{userName}}</span>
			</p>
			<p class='userInfo'>
				<span><%=rb.getString("GuDingDianHua")%>:&nbsp;</span>
				<span>{{phone}}</span>
			</p>
			<div class='userInfo' style='display:flex'>
				<span><%=rb.getString("YongHuZu")%>:&nbsp;</span>
				<p>
					<span style='color:#333;width:auto' class='groupInfo' v-for='item in groupItem'>{{item}}</span>
				</p>
			</div>
		</div>
		<div style='flex:1;height:120px;margin-top:10px;height:auto'>
			<p class='userInfo'>
				<span><%=rb.getString("YouXiang")%>:&nbsp;</span>
				<span>{{email}}</span>
			</p>
			<p class='userInfo'>
				<span><%=rb.getString("YouXiaoQiSuoDing")%>:&nbsp;</span>
				<span>{{lock}}</span>
			</p>
			<p class='userInfo'>
				<span><%=rb.getString("DaoQiShiJian")%>:&nbsp;</span>
				<span>{{date}}</span>
				<span v-if='showDay' style='font-size:14px;color:#4D84FF'>(<%=rb.getString("ShengYuShiJian")%>:{{remainDays}})</span>
			</p>
		</div>
	</div>
	<p class='userInfo'>
		<span><%=rb.getString("LaiYuan")%>:&nbsp;</span>
		<span>{{source}}</span>
	</p>
	<p class='userInfo'>
		<span><%=rb.getString("MiaoShu")%>:&nbsp;</span>
		<span>{{desc}}</span>
	</p>
</div>
<script>
	new Vue({
		el:'#viewUserInfo',
		data(){
			return{
				userName:'',
				phone:'',
				email:'',
				lock:'',
				date:'',
				desc:'',
				source:'',
				remainDays:'',
				showDay:'',
				groupItem:[]
			}
		},
		methods:{
			init(){
				var vm = this;
				if(userVue.userRowData.group_name == null){
					
				}else{
					vm.groupItem = userVue.userRowData.group_name.split(",");
				}
				axios.post("${ctx}/system/sysuser/getUserDetil.action",stringify({user_id:userVue.userRowData.id,timeZone:timeZone})).then(function(response){
					var data = response.data;
					vm.userName = data.user_code;
					vm.email = data.user_email;
					vm.phone = data.user_cell;
					vm.desc = data.user_desc;
					vm.source = data.source;
					vm.date = data.expites_date;
					var lock = data.lock_status;
					if(lock == '0'){
						vm.lock = '<%=rb.getString("JieSuo")%>'
					}else{
						vm.lock = '<%=rb.getString("YouXiaoQiSuoDing")%>'
					}
					vm.remainDays = data.remain_day;
					 if(data.expites_date == ""){
						vm.showDay = false;
					}else{
						vm.showDay = true;
					}
				})
			}
		},
		mounted(){
			this.init();
		}
	})
</script>